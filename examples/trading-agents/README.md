# TradingAgents → tossctl Bridge

[TradingAgents](https://github.com/TauricResearch/TradingAgents)의 멀티 에이전트 분석 결과를 `tossctl`로 실행하는 브릿지 스크립트.

## 흐름

```
TradingAgents (LLM 멀티 에이전트 분석)
    ↓ BUY / OVERWEIGHT / HOLD / UNDERWEIGHT / SELL
예산 배분 (BUY=균등, OVERWEIGHT=70%, 나머지=0)
    ↓
tossctl order place --fractional --amount <KRW>
```

## 사전 준비

```bash
# 1. TradingAgents 설치
pip install tradingagents

# 2. tossctl 설치 및 로그인
curl -fsSL https://raw.githubusercontent.com/JungHoonGhae/tossinvest-cli/main/install.sh | sh
tossctl auth login

# 3. config.json에서 거래 허용
# place, fractional, allow_live_order_actions → true

# 4. 거래 권한 부여
tossctl order permissions grant --ttl 600

# 5. LLM API 키 설정
export ANTHROPIC_API_KEY=...  # 또는 OPENAI_API_KEY
```

## 사용법

```bash
# dry-run: TradingAgents 분석 없이 tossctl preview만 테스트
python bridge.py --symbols TSLL --budget 50000 --dry-run

# 분석만 (주문 안 함)
python bridge.py --symbols TSLL NVDA AAPL --budget 100000

# 분석 + 실제 주문
python bridge.py --symbols TSLL NVDA AAPL --budget 100000 --execute

# OpenAI 사용
python bridge.py --symbols TSLL --budget 50000 --provider openai --model gpt-5-mini
```

## 신호별 동작

| 신호 | 동작 |
|------|------|
| BUY | 균등 배분 매수 |
| OVERWEIGHT | 70% 비중 매수 |
| HOLD | 매수 안 함 |
| UNDERWEIGHT | 매수 안 함 |
| SELL | 매수 안 함 (매도 미구현) |

## 주의

- 이 스크립트는 실험/학습 목적입니다
- TradingAgents의 분석이 수익을 보장하지 않습니다
- 소액으로 테스트하세요
- `--execute` 없이 먼저 preview를 확인하세요
