#!/usr/bin/env python3
"""
TradingAgents → tossctl bridge

TradingAgents의 분석 결과(BUY/SELL/HOLD)를 받아 tossctl로 주문을 실행합니다.
소수점 매수(fractional)로 소액 분산 투자합니다.

사용법:
    python bridge.py --symbols TSLL NVDA AAPL --budget 100000
    python bridge.py --symbols TSLL --budget 50000 --dry-run
    python bridge.py --symbols TSLL NVDA --budget 100000 --execute
"""

import argparse
import json
import os
import shutil
import subprocess
import sys
from datetime import date
from dataclasses import dataclass


def find_tossctl() -> str:
    """tossctl 바이너리 경로를 찾는다."""
    found = shutil.which("tossctl")
    if found:
        return found
    # 일반적인 설치 경로들
    script_dir = os.path.dirname(os.path.abspath(__file__))
    for candidate in [
        "/usr/local/bin/tossctl",
        os.path.expanduser("~/go/bin/tossctl"),
        os.path.join(script_dir, "..", "..", "bin", "tossctl"),  # 개발 빌드
    ]:
        if os.path.isfile(candidate):
            return candidate
    print("Error: tossctl을 찾을 수 없습니다. 설치 후 다시 시도하세요.", file=sys.stderr)
    sys.exit(1)


TOSSCTL = find_tossctl()


@dataclass
class Signal:
    symbol: str
    decision: str  # BUY, OVERWEIGHT, HOLD, UNDERWEIGHT, SELL
    full_report: str


def analyze(symbol: str, config: dict) -> Signal:
    """TradingAgents로 종목 분석."""
    from tradingagents.graph.trading_graph import TradingAgentsGraph

    ta = TradingAgentsGraph(debug=False, config=config)
    state, decision = ta.propagate(symbol, str(date.today()))

    return Signal(
        symbol=symbol,
        decision=decision.strip().upper(),
        full_report=state.get("final_trade_decision", ""),
    )


def allocate_budget(signals: list[Signal], total_budget: int) -> dict[str, int]:
    """신호별 예산 배분. BUY=전체/n, OVERWEIGHT=전체/n*0.7, 나머지=0."""
    buy_signals = [s for s in signals if s.decision in ("BUY", "OVERWEIGHT")]

    if not buy_signals:
        return {}

    per_stock = total_budget // len(buy_signals)
    allocation = {}

    for s in buy_signals:
        if s.decision == "BUY":
            allocation[s.symbol] = per_stock
        elif s.decision == "OVERWEIGHT":
            allocation[s.symbol] = int(per_stock * 0.7)

    return allocation


def tossctl(*args) -> dict:
    """tossctl 실행 후 JSON 파싱."""
    cmd = [TOSSCTL] + list(args) + ["--output", "json"]
    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"  tossctl error: {result.stderr.strip()}", file=sys.stderr)
        return {"error": result.stderr.strip()}

    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return {"raw": result.stdout.strip()}


def preview_order(symbol: str, amount: int) -> dict:
    """주문 미리보기."""
    return tossctl(
        "order", "preview",
        "--symbol", symbol,
        "--side", "buy",
        "--fractional",
        "--amount", str(amount),
        "--qty", "0",
    )


def place_order(symbol: str, amount: int, confirm_token: str) -> dict:
    """실제 주문 실행."""
    return tossctl(
        "order", "place",
        "--symbol", symbol,
        "--side", "buy",
        "--fractional",
        "--amount", str(amount),
        "--qty", "0",
        "--execute",
        "--dangerously-skip-permissions",
        "--confirm", confirm_token,
    )


def main():
    parser = argparse.ArgumentParser(description="TradingAgents → tossctl bridge")
    parser.add_argument("--symbols", nargs="+", required=True, help="분석할 종목 (예: TSLL NVDA)")
    parser.add_argument("--budget", type=int, required=True, help="총 투자 예산 (KRW)")
    parser.add_argument("--execute", action="store_true", help="실제 주문 실행 (없으면 preview만)")
    parser.add_argument("--dry-run", action="store_true", help="TradingAgents 분석 없이 tossctl preview만 테스트")
    parser.add_argument("--provider", default="anthropic", help="LLM provider (default: anthropic)")
    parser.add_argument("--model", default="claude-sonnet-4-6", help="LLM model (default: claude-sonnet-4-6)")
    args = parser.parse_args()

    if args.budget < 1000 * len(args.symbols):
        print(f"Error: 종목당 최소 1,000원 필요 (총 {1000 * len(args.symbols):,}원)", file=sys.stderr)
        sys.exit(1)

    # --- 1. 분석 ---
    print(f"=== TradingAgents Bridge ===")
    print(f"종목: {', '.join(args.symbols)}")
    print(f"예산: {args.budget:,}원")
    print()

    if args.dry_run:
        # dry-run: 분석 건너뛰고 모든 종목 BUY로 가정
        signals = [Signal(symbol=s, decision="BUY", full_report="[dry-run]") for s in args.symbols]
        print("[dry-run] 분석 건너뜀 — 모든 종목 BUY로 가정")
    else:
        from tradingagents.default_config import DEFAULT_CONFIG

        config = DEFAULT_CONFIG.copy()
        config["llm_provider"] = args.provider
        config["deep_think_llm"] = args.model
        config["quick_think_llm"] = args.model
        config["max_debate_rounds"] = 1
        config["data_vendors"] = {
            "core_stock_apis": "yfinance",
            "technical_indicators": "yfinance",
            "fundamental_data": "yfinance",
            "news_data": "yfinance",
        }

        signals = []
        for symbol in args.symbols:
            print(f"[분석] {symbol}...", end=" ", flush=True)
            try:
                signal = analyze(symbol, config)
                signals.append(signal)
                print(f"→ {signal.decision}")
            except Exception as e:
                print(f"→ ERROR: {e}")
                signals.append(Signal(symbol=symbol, decision="HOLD", full_report=str(e)))

    # --- 2. 결과 요약 ---
    print()
    print("=== 분석 결과 ===")
    for s in signals:
        print(f"  {s.symbol}: {s.decision}")

    # --- 3. 예산 배분 ---
    allocation = allocate_budget(signals, args.budget)

    if not allocation:
        print()
        print("매수 신호 없음. 종료.")
        sys.exit(0)

    print()
    print("=== 예산 배분 ===")
    for symbol, amount in allocation.items():
        print(f"  {symbol}: {amount:,}원")

    # --- 4. Preview ---
    print()
    print("=== 주문 Preview ===")
    previews = {}
    for symbol, amount in allocation.items():
        preview = preview_order(symbol, amount)
        previews[symbol] = preview
        print(f"  {symbol}: {json.dumps(preview, ensure_ascii=False, indent=2)}")

    # --- 5. Execute ---
    if not args.execute:
        print()
        print("--execute 없이 실행됨. 실제 주문은 하지 않습니다.")
        print("실행하려면: python bridge.py --symbols ... --budget ... --execute")
        sys.exit(0)

    print()
    print("=== 주문 실행 ===")
    for symbol, amount in allocation.items():
        preview = previews.get(symbol, {})
        token = preview.get("confirm_token", "")

        if not token:
            print(f"  {symbol}: confirm_token 없음 — skip")
            continue

        if not preview.get("mutation_ready", False):
            print(f"  {symbol}: mutation_ready=false — skip (config 확인 필요)")
            continue

        print(f"  {symbol}: 주문 중 ({amount:,}원)...", end=" ", flush=True)
        result = place_order(symbol, amount, token)

        if "error" in result:
            print(f"→ FAIL: {result['error']}")
        else:
            status = result.get("status", "unknown")
            print(f"→ {status}")


if __name__ == "__main__":
    main()
