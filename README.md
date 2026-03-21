<div align="center">
  <h1>tossinvest-cli</h1>
  <p>토스증권 웹 세션을 재사용해 조회와 제한된 거래 기능을 터미널에서 다루기 위한 비공식 CLI입니다.</p>
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
> 현재 거래 기능은 live 검증이 쌓인 좁은 베타입니다. 설치 직후에는 모든 거래 기능이 기본적으로 꺼져 있고, 사용자가 `config.json`에서 기능별로 직접 허용해야만 실행할 수 있습니다.

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

## tossinvest-cli란

`tossinvest-cli`는 토스증권 웹 세션을 재사용해 아래 작업을 CLI에서 다룰 수 있게 만든 도구입니다.

- 계좌 상태 읽기
- 시세와 미체결 주문 확인
- 주문 전 확인
- 제한된 범위의 주문 실행
- 다른 자동화 흐름으로 결과 전달

현재는 조회 기능과 좁은 거래 베타 범위를 중심으로 구현되어 있습니다.

## 현재 상태

- 조회 기능은 일상적인 스크립트/자동화 용도로 바로 쓸 수 있습니다.
- 거래 기능은 `US/KR buy/sell limit / KRW / non-fractional` 슬라이스에 한해 live 검증이 쌓여 있습니다. sell은 `trading.sell=true`, 국내주식은 `trading.kr=true` 추가 설정이 필요합니다.
- 기본 동작은 안전 우선입니다.
  - config 허용
  - temporary permission
  - `preview`
  - `--execute`
  - `--dangerously-skip-permissions`
  - `--confirm`
- 브로커 분기가 나오면 설명하고 멈추는 것이 기본입니다.
  - 현재 예외는 `dangerous_automation.accept_fx_consent=true`일 때의 post-prepare FX confirmation branch뿐입니다.

## 이런 경우에 잘 맞습니다

- 자산 현황을 스크립트에서 주기적으로 확인하고 싶을 때
- 미체결 주문과 시세를 `json`으로 뽑아 다른 시스템에 넘기고 싶을 때
- 조건 계산과 실제 실행 단계를 분리하고 싶을 때
- 자동화 도구가 토스증권 작업을 명령 단위로 호출할 수 있게 만들고 싶을 때

## 어떤 문제를 줄여주는가

| 직접 웹에서 할 때 | `tossinvest-cli`를 쓰면 |
|---|---|
| 계좌, 시세, 미체결 주문을 반복해서 열어 봐야 한다 | 조회 명령을 스크립트와 터미널에서 바로 호출할 수 있다 |
| 데이터를 다른 도구로 넘기기 어렵다 | `json` 출력으로 후속 자동화에 연결할 수 있다 |
| 주문 전 확인과 실제 실행을 한 흐름에서 섞기 쉽다 | `preview`와 실행 단계를 분리할 수 있다 |
| 브라우저 세션이 살아 있어도 자동화 경로가 없다 | 웹 세션을 재사용하는 CLI 흐름을 만들 수 있다 |

## Config

기본 설정 파일 경로는 `<config dir>/config.json`입니다. 파일이 없어도 CLI는 동작하지만, 이 경우 거래 기능은 모두 비활성 상태로 동작합니다.

```bash
tossctl config init
tossctl config show
```

생성되는 기본 설정은 아래와 같습니다.

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

`grant`, `place`, `sell`, `kr`, `cancel`, `amend`는 기능별 허용 여부입니다. `sell`은 `place`와 함께 켜야 매도 주문이 가능합니다. `kr`은 `place`와 함께 켜야 국내주식 주문이 가능합니다. `allow_live_order_actions`는 실제 계좌에 영향을 주는 주문 액션 자체를 허용할지 정하고, `dangerous_automation`은 어떤 위험한 브로커 분기를 자동 진행할 수 있게 둘지 정합니다.

현재 실제 handler가 연결된 dangerous automation은 아래 하나입니다.

- `trading.dangerous_automation.accept_fx_consent`
  - `prepare`는 성공했지만 `needExchange > 0`이라서 웹과 같은 FX confirmation branch가 뜨는 경우
  - `false`면 CLI가 설명하고 멈춥니다.
  - `true`면 확인된 웹 흐름과 같은 방식으로 `order/create`를 계속 진행합니다.

나머지 dangerous automation key는 설정 표면과 정책 공간은 열려 있지만, 아직 같은 수준으로 닫힌 실행 handler가 아닐 수 있습니다.

## 지원 범위

### 지금 바로 되는 것

- 브라우저 로그인 기반 세션 저장과 재사용
- 계좌, 포트폴리오, 미체결 주문, 관심종목, 시세 조회
- `orders completed`, `order show <id>`로 pending/completed 주문 추적
- 제한된 거래 베타
  - `미국주식`
  - `매수` / `매도` (매도는 `trading.sell=true` 필요)
  - `소수점 주문` (US 시장가, `trading.fractional=true` 필요)
  - `지정가`
  - `KRW`
  - `비소수점`
- 당일 미체결 주문 취소
- 거래 분석 문서와 민감정보를 정리한 fixture 관리

### live 검증이 끝난 범위

- `order preview`
- `order place` for `US/KR buy/sell limit / KRW / non-fractional` (sell requires `trading.sell=true`, KR requires `trading.kr=true`)
- `order cancel` for same-day pending orders
- `orders completed`
- `order show <id>`
- local lineage fallback for `order show <old-id>` after same-machine `cancel` or `amend` rollover
- step-by-step operator guidance when `order place` is blocked by funding or FX-consent branches
- post-prepare FX branch stop after successful `prepare`
- `dangerous_automation.accept_fx_consent` for the known post-prepare FX confirmation branch

### 아직 더 필요한 것

- `order amend` 재검증
- `amend` interactive auth branch 정리
- `시장가` (비소수점)

## 이 프로젝트가 하지 않는 것

| 하지 않는 것 | 설명 |
|---|---|
| 공식 API SDK 제공 | 토스증권 공식 API나 공식 지원 SDK를 제공하는 프로젝트가 아닙니다. |
| 범용 트레이딩 클라이언트 | 모든 주문 유형과 시장을 완전히 지원하지 않습니다. |
| 무제한 자동 매매 | 안전장치 없이 바로 실행되는 자동 매매 도구를 목표로 하지 않습니다. |

## 설치

### Homebrew

macOS에서는 Homebrew 설치를 기본 경로로 생각하고 있습니다.

```bash
brew tap JungHoonGhae/tossinvest-cli
brew install tossctl
```

설치 후에는 먼저 환경을 확인합니다.

```bash
tossctl version
tossctl doctor
tossctl config show
tossctl auth doctor
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
tossctl version
tossctl doctor
tossctl config show

tossctl auth login
tossctl auth doctor
tossctl auth status
tossctl auth logout

tossctl account list
tossctl account summary
tossctl portfolio positions
tossctl portfolio allocation
tossctl orders list
tossctl orders completed
tossctl watchlist list
tossctl quote get <symbol>
```

### 거래

```bash
tossctl order preview
tossctl order show <id>
tossctl config init
tossctl config show
tossctl order place
tossctl order cancel
tossctl order amend
tossctl order permissions status
tossctl order permissions grant --ttl 300
tossctl order permissions revoke
```

## 주문 예시

아래는 `TSLL` 1주를 `500원` 지정가로 미리보기한 뒤 실제 주문하는 흐름입니다.

```bash
tossctl config init
# edit config.json and set trading.grant/place/allow_live_order_actions to true
# for sell orders, also set trading.sell to true
# for KR stocks, also set trading.kr to true

tossctl order preview \
  --symbol TSLL \
  --market us \
  --side buy \
  --type limit \
  --qty 1 \
  --price 500 \
  --currency-mode KRW \
  --output json

tossctl order permissions grant --ttl 300

tossctl order place \
  --symbol TSLL \
  --market us \
  --side buy \
  --type limit \
  --qty 1 \
  --price 500 \
  --currency-mode KRW \
  --execute \
  --dangerously-skip-permissions \
  --confirm <preview-token> \
  --output json

tossctl orders list --output json
```

만약 `1000 KRW`처럼 post-prepare FX confirmation branch가 나오는 입력까지 자동 진행하려면, config에 아래 값을 추가로 켭니다.

```json
{
  "trading": {
    "dangerous_automation": {
      "accept_fx_consent": true
    }
  }
}
```

## 거래 안전장치

거래 기능은 기본적으로 여러 단계 확인을 거치게 되어 있습니다.

- `config.json`에서 기능별 허용
- `config.json`에서 `allow_live_order_actions` 허용
- `order preview`
- `order permissions grant`
- `--execute`
- `--dangerously-skip-permissions`
- `--confirm`

금융 자동화에서는 편의성보다 오작동 방지가 먼저라고 보고 있습니다.

## 주문 ref rollover

`amend`나 `cancel` 이후에는 브로커 쪽 주문 ref가 새 값으로 바뀔 수 있습니다.

- mutation 결과에 `original_order_id`와 `current_order_id`가 함께 나오면 rollover가 발생한 것입니다.
- 같은 로컬 `config dir`에서 실행한 mutation이면 `tossctl order show <old-id>`가 local lineage cache를 통해 새 ref를 다시 찾을 수 있습니다.
- lineage cache는 `<config dir>/trading-lineage.json`에 저장됩니다.
- broker의 completed-history 반영이 늦어서 mutation 시점에 `current_order_id`를 못 잡은 cancel도, 나중에 `order show <old-id>`가 delayed completed-history row를 다시 조회해 같은 config dir 안에서 lineage cache를 갱신할 수 있습니다.
- 다만 같은 조건의 canceled row가 여러 개면 추정하지 않고 실패하므로, 그런 경우에는 먼저 `orders completed`에서 새 ref를 확인해야 합니다.
- 다른 머신이나 다른 `--config-dir`에서 실행한 주문까지 추적하는 기능은 아직 아닙니다.

## 개발

```bash
make tidy
make fmt
make build
make test
```

`auth-helper`는 브라우저 로그인만 담당합니다. 토스증권 도메인 로직은 Go CLI 쪽에 남겨두고, 브라우저 자동화는 분리해 유지합니다.

## FAQ

**누가 쓰기 좋은가요?**  
토스증권 조회를 스크립트에 넣고 싶거나, 주문 전 확인과 제한된 주문 흐름을 CLI로 다루고 싶은 사용자에게 맞습니다.

**바로 주문까지 가능한가요?**  
일부 범위만 베타로 지원합니다. 현재 live 검증이 끝난 건 `US buy limit / KRW / non-fractional` 기준의 `place`, 당일 pending `cancel`, `orders completed`, `order show <id>` 기반 상태 조회입니다. `cancel`과 `amend` 후 ref rollover는 same-machine local lineage cache로 다시 찾을 수 있고, delayed cancel rollover도 `order show <old-id>`가 later completed-history lookup으로 복구할 수 있습니다. 다만 ambiguous candidate가 생기면 수동 확인이 필요합니다. `order place`가 funding 분기에 막히면 CLI가 단계별 행동과 재시도 명령을 안내합니다. FX confirmation branch는 기본적으로 설명하고 멈추지만, `trading.dangerous_automation.accept_fx_consent=true`를 켜면 현재 확인된 경로에 한해 자동 진행할 수 있습니다. `amend` 자체는 아직 더 많은 live 검증이 필요합니다. 거래 기능은 먼저 `config.json`에서 해당 액션을 직접 허용해야 합니다.

**공식 API인가요?**  
아닙니다. 토스증권 공식 제품이 아니고, 웹 내부 API를 재사용하는 비공식 프로젝트입니다.

**왜 Playwright가 필요한가요?**  
로그인 세션을 브라우저 흐름으로 확보하기 위해 필요합니다. 실제 조회와 거래 로직은 Go CLI 쪽에 구현되어 있습니다.

## 문서

- [`docs/architecture.md`](docs/architecture.md)
- [`docs/configuration.md`](docs/configuration.md)
- [`docs/reverse-engineering/`](docs/reverse-engineering/)
- [`docs/trading/`](docs/trading/)
- [`auth-helper/README.md`](auth-helper/README.md)

## 상태 확인 예시

```bash
tossctl orders completed --market us --output json
tossctl order show <order-id> --market us --output json
```

## 로컬 저장 경로

- config dir: `$(os.UserConfigDir)/tossctl`
- config file: `<config dir>/config.json`
- cache dir: `$(os.UserCacheDir)/tossctl`
- session file: `<config dir>/session.json`
- permission file: `<config dir>/trading-permission.json`
- lineage file: `<config dir>/trading-lineage.json`

개발 중에는 아래 플래그로 경로를 덮어쓸 수 있습니다.

- `--config-dir`
- `--session-file`

## Contributing

버그 제보와 PR은 환영합니다.

## Support

도움이 되었다면 유지보수에 힘을 보태 주세요.

<a href="https://www.buymeacoffee.com/lucas.ghae">
  <img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" height="50">
</a>

## License

MIT
