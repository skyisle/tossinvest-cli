<div align="center">
  <h1>tossinvest-cli</h1>
  <p>토스증권 웹 세션을 재사용해 조회와 거래를 터미널에서 다루기 위한 비공식 CLI입니다.</p>
  <p>실행 바이너리는 <code>tossctl</code>입니다.</p>
</div>

<p align="center">
  <a href="#quick-start"><strong>Quick Start</strong></a> ·
  <a href="#지원-범위"><strong>지원 범위</strong></a> ·
  <a href="#명령-표면"><strong>명령 표면</strong></a> ·
  <a href="#faq"><strong>FAQ</strong></a> ·
  <a href="#문서"><strong>문서</strong></a>
</p>

<p align="center">
  <a href="https://github.com/JungHoonGhae/tossinvest-cli/stargazers"><img src="https://img.shields.io/github/stars/JungHoonGhae/tossinvest-cli" alt="GitHub stars" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="MIT License" /></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8.svg" alt="Go" /></a>
  <a href="https://github.com/JungHoonGhae/tossinvest-cli"><img src="https://img.shields.io/badge/status-beta-orange.svg" alt="Status Beta" /></a>
  <a href="https://github.com/JungHoonGhae/tossinvest-cli/actions/workflows/ci.yml"><img src="https://github.com/JungHoonGhae/tossinvest-cli/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
</p>

> [!WARNING]
> 이 프로젝트는 토스증권 공식 제품이 아닙니다. 웹 내부 API는 예고 없이 바뀔 수 있고, 잘못 쓰면 실제 계좌에 영향을 줄 수 있습니다.

> [!IMPORTANT]
> 거래 기능은 설치 직후 모두 꺼져 있습니다. `config.json`에서 기능별로 직접 허용해야만 실행됩니다.

<div align="center">
<table>
  <tr>
    <td align="center"><strong>Works with</strong></td>
    <td align="center"><img src="docs/assets/logos/openclaw.svg" width="32" alt="OpenClaw" /><br /><sub>OpenClaw</sub></td>
    <td align="center"><img src="docs/assets/logos/claude.svg" width="32" alt="Claude Code" /><br /><sub>Claude Code</sub></td>
    <td align="center"><img src="docs/assets/logos/codex.svg" width="32" alt="Codex" /><br /><sub>Codex</sub></td>
    <td align="center"><img src="docs/assets/logos/cursor.svg" width="32" alt="Cursor" /><br /><sub>Cursor</sub></td>
    <td align="center"><img src="docs/assets/logos/bash.svg" width="32" alt="Bash" /><br /><sub>Bash</sub></td>
    <td align="center"><img src="docs/assets/logos/http.svg" width="32" alt="HTTP" /><br /><sub>HTTP</sub></td>
  </tr>
</table>
</div>

## Quick Start

### For Human

```bash
brew tap JungHoonGhae/tossinvest-cli
brew install tossctl

tossctl version
tossctl doctor
tossctl config show
tossctl auth doctor
tossctl auth login
tossctl account summary --output json
```

`auth login`까지 쓰려면 Homebrew Python에 Playwright와 Chromium을 준비해야 합니다.

```bash
PY="$(brew --prefix python@3.11)/bin/python3.11"
"$PY" -m pip install playwright
"$PY" -m playwright install chromium
```

### For Agent

```text
Install tossinvest-cli with Homebrew, run `tossctl doctor` and `tossctl auth doctor`,
complete browser login with `tossctl auth login`, then use read-only commands first.
Trading actions stay disabled until config.json explicitly allows them.
Only use `tossctl order preview` before any trading mutation.
```

## 지원 범위

### 조회 (읽기 전용)

| 기능 | 커맨드 | US | KR |
|------|--------|:--:|:--:|
| 계좌 목록 / 요약 | `account list`, `account summary` | O | O |
| 포트폴리오 | `portfolio positions`, `portfolio allocation` | O | O |
| 시세 | `quote get <symbol>`, `quote batch <sym> [sym...]` | O | O |
| 미체결 주문 | `orders list` | O | O |
| 체결 내역 | `orders completed --market us\|kr\|all` | O | O |
| 단건 주문 조회 | `order show <id>` | O | O |
| 관심 종목 | `watchlist list` | O | O |
| CSV 내보내기 | `export positions --market`, `export orders --market` | O | O |

### 거래

| 기능 | 커맨드 | 필요 config |
|------|--------|-------------|
| 지정가 매수 (US/KR) | `order place --side buy --price <KRW>` | `place` |
| 지정가 매도 (US/KR) | `order place --side sell --price <KRW>` | `place` + `sell` |
| 국내주식 거래 | `order place --market kr` | `place` + `kr` |
| 소수점 매수 (US) | `order place --fractional --amount <KRW>` | `place` + `fractional` |
| 주문 취소 | `order cancel --order-id <id>` | `cancel` |
| 주문 정정 | `order amend --order-id <id>` | `amend` |
| 거래 권한 관리 | `order permissions grant\|status\|revoke` | `grant` |

모든 거래는 `allow_live_order_actions=true`도 필요합니다. 소수점 주문은 시장가(market order)로 자동 전환되며, 금액(KRW) 기반입니다.

### Safety Model

```
config.json 허용 → permissions grant (TTL) → preview → --execute
  → --dangerously-skip-permissions → --confirm <token>
```

6단계 게이트. 거래 기능은 기본 전부 꺼져 있고, 하나씩 열어야 실행 가능.

## Config

```bash
tossctl config init
tossctl config show
```

```json
{
  "$schema": "https://raw.githubusercontent.com/JungHoonGhae/tossinvest-cli/main/schemas/config.schema.json",
  "schema_version": 2,
  "trading": {
    "grant": false,
    "place": false,
    "sell": false,
    "kr": false,
    "fractional": false,
    "cancel": false,
    "amend": false,
    "allow_live_order_actions": false,
    "dangerous_automation": {
      "complete_trade_auth": false,
      "accept_product_ack": false,
      "accept_fx_consent": false
    }
  }
}
```

| 필드 | 설명 |
|------|------|
| `grant` | `order permissions grant` 허용 |
| `place` | `order place` 허용 |
| `sell` | 매도 주문 허용 (`place`도 필요) |
| `kr` | 국내주식 거래 허용 (`place`도 필요) |
| `fractional` | 소수점 주문 허용 (`place`도 필요, US 시장가만) |
| `cancel` | `order cancel` 허용 |
| `amend` | `order amend` 허용 |
| `allow_live_order_actions` | 실계좌에 영향을 주는 주문 액션 허용 |
| `accept_fx_consent` | post-prepare FX confirmation 자동 진행 |

## 주문 예시

### 지정가 매수 (US)

```bash
tossctl config init
# config.json: grant, place, allow_live_order_actions → true

tossctl order preview \
  --symbol TSLL --side buy --qty 1 --price 18000 --output json

tossctl order permissions grant --ttl 300

tossctl order place \
  --symbol TSLL --side buy --qty 1 --price 18000 \
  --execute --dangerously-skip-permissions --confirm <token> \
  --output json
```

### 소수점 매수 (US, 금액 기반)

```bash
# config.json: grant, place, fractional, allow_live_order_actions → true

tossctl order preview \
  --symbol TSLL --side buy --fractional --amount 1000 --qty 0 --output json

tossctl order place \
  --symbol TSLL --side buy --fractional --amount 1000 --qty 0 \
  --execute --dangerously-skip-permissions --confirm <token> \
  --output json
```

### 국내주식 매수

```bash
# config.json: grant, place, kr, allow_live_order_actions → true

tossctl order place \
  --symbol 005930 --market kr --side buy --qty 1 --price 200000 \
  --execute --dangerously-skip-permissions --confirm <token>
```

### 매도

```bash
# config.json: sell → true (추가)

tossctl order place \
  --symbol TSLL --side sell --qty 1 --price 18000 \
  --execute --dangerously-skip-permissions --confirm <token>
```

### 다종목 시세

```bash
tossctl quote batch TSLL 005930 GOOG VOO --output table
```

## 이 프로젝트가 하지 않는 것

| 하지 않는 것 | 설명 |
|---|---|
| 공식 API SDK 제공 | 토스증권 공식 API나 공식 지원 SDK를 제공하는 프로젝트가 아닙니다. |
| 범용 트레이딩 클라이언트 | 모든 주문 유형과 시장을 완전히 지원하지 않습니다. |
| 무제한 자동 매매 | 안전장치 없이 바로 실행되는 자동 매매 도구를 목표로 하지 않습니다. |

## 설치

### Homebrew

```bash
brew tap JungHoonGhae/tossinvest-cli
brew install tossctl
```

### From source

```bash
git clone https://github.com/JungHoonGhae/tossinvest-cli.git
cd tossinvest-cli
make build

cd auth-helper
python3 -m pip install -e .
python3 -m playwright install chromium
```

## 명령 표면

### 조회

```bash
tossctl account list
tossctl account summary
tossctl portfolio positions
tossctl portfolio allocation
tossctl orders list
tossctl orders completed --market us|kr|all
tossctl order show <id>
tossctl quote get <symbol>
tossctl quote batch <symbol> [symbol...]
tossctl watchlist list
tossctl export positions --market us|kr|all
tossctl export orders --market us|kr|all
```

### 거래

```bash
tossctl order preview --symbol <sym> --side <buy|sell> --qty <n> --price <krw>
tossctl order preview --symbol <sym> --side buy --fractional --amount <krw> --qty 0
tossctl order place ...flags... --execute --dangerously-skip-permissions --confirm <token>
tossctl order cancel --order-id <id> --symbol <sym> ...
tossctl order amend --order-id <id> ...
tossctl order permissions grant --ttl 300
tossctl order permissions status
tossctl order permissions revoke
```

### 시스템

```bash
tossctl version
tossctl doctor
tossctl config init
tossctl config show
tossctl auth login
tossctl auth status
tossctl auth doctor
tossctl auth logout
```

## 주문 ref rollover

`amend`나 `cancel` 이후 브로커 쪽 주문 ref가 바뀔 수 있습니다.

- `tossctl order show <old-id>`가 local lineage cache를 통해 새 ref를 추적합니다.
- lineage cache: `<config dir>/trading-lineage.json`
- 같은 조건의 canceled row가 여러 개면 수동 확인이 필요합니다.

## 개발

```bash
make build
make test
make fmt
make tidy
```

## FAQ

**바로 주문까지 가능한가요?**
US/KR 지정가 매수/매도, US 소수점 매수, 당일 미체결 취소가 live 검증되어 있습니다. `amend`는 추가 검증이 필요합니다. 모든 거래는 `config.json`에서 해당 액션을 허용한 뒤에만 실행됩니다.

**공식 API인가요?**
아닙니다. 웹 내부 API를 재사용하는 비공식 프로젝트입니다.

**왜 Playwright가 필요한가요?**
로그인 세션을 브라우저 흐름으로 확보하기 위해 필요합니다. 조회/거래 로직은 Go CLI에 구현되어 있습니다.

## 문서

- [`docs/architecture.md`](docs/architecture.md)
- [`docs/configuration.md`](docs/configuration.md)
- [`docs/reverse-engineering/`](docs/reverse-engineering/)
- [`docs/trading/`](docs/trading/)
- [`auth-helper/README.md`](auth-helper/README.md)

## 로컬 저장 경로

| 경로 | 설명 |
|------|------|
| `<config dir>/config.json` | 거래 설정 |
| `<config dir>/session.json` | 브라우저 세션 |
| `<config dir>/trading-permission.json` | 임시 거래 권한 |
| `<config dir>/trading-lineage.json` | 주문 ref 추적 |

`--config-dir`, `--session-file` 플래그로 경로를 덮어쓸 수 있습니다.

## Contributing

버그 제보와 PR은 환영합니다.

## Support

도움이 되었다면 유지보수에 힘을 보태 주세요.

<a href="https://www.buymeacoffee.com/lucas.ghae">
  <img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" height="50">
</a>

## License

MIT
