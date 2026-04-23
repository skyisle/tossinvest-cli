# Changelog

All notable changes to this project will be documented in this file.

## [0.4.1] - 2026-04-23

보안 하드닝 릴리즈. 전체 시스템을 점검하여 기능 영향 없이 좁힐 수 있는 부분만 적용.

### Security
- **Intermediate storage-state 파일 권한 0o600 고정** — Python helper가 저장하는 `~/Library/Caches/tossctl/auth/playwright-storage-state.json`은 이전엔 default umask(보통 0o644)를 따라 쓰였음. `os.open(..., O_CREAT|O_WRONLY|O_TRUNC, 0o600)` + `os.fchmod(0o600)`로 기존 파일 유무와 관계없이 항상 소유자만 읽도록 변경. Go CLI가 session.json으로 복사하기 전 찰나의 창에 같은 호스트의 다른 로컬 사용자가 전체 쿠키 세트를 읽을 수 있던 공격면 차단.
- **`--qr-output` PNG 권한 강제 0o600** — 동일 `fchmod` 패턴 적용. 기존 파일이 이미 0o644로 존재해도 overwrite 시 명시적으로 좁힘.
- **Intermediate storage-state 파일 로그인 성공 후 자동 삭제** — `LoginWith`가 `ImportPlaywrightState` 성공 직후 `os.Remove(result.StorageStatePath)` 호출. 같은 쿠키의 중복 사본이 cache dir에 무기한 남던 문제 해결 (`auth logout`은 session.json만 지웠음). 사용자가 직접 부르는 `auth import-playwright-state <path>`는 경유하지 않으므로 외부 파일은 그대로 유지.
- **tossctl 상태 디렉토리 권한 0o755 → 0o700** — `session/store.go`, `config/service.go`, `permissions/service.go`, `orderlineage/service.go`의 `os.MkdirAll` 모드 좁힘. macOS `~/Library/Application Support`는 부모 디렉토리가 이미 0o700이라 영향 미미하지만 Linux/CI 환경에선 같은 호스트의 다른 사용자가 `tossctl/` 디렉토리 목록을 열람 가능하던 문제 차단.
- **`AuthError`에서 응답 본문(Body) 필드 제거** — `wts-api` / `wts-cert-api`의 401/403 응답 본문엔 CSRF 진단이나 세션 식별자 조각이 포함될 수 있음. 현재 어떤 caller도 `AuthError.Body`를 읽지 않지만, 향후 `%+v` 디버그 로그나 에러 값 직렬화 시 유출될 수 있어 필드 자체를 제거. `StatusError.Body`는 trading broker 메시지 분류에 실제로 사용되므로 유지.

## [0.4.0] - 2026-04-23

### Added
- **거래내역 ledger + cash overview** — `tossctl transactions list --market kr|us`로 매매, 입출금, 배당, 주식 입출고를 조회. `--from/--to`, `--filter all|trade|cash|inout|cash-alt`, `--all` 페이지네이션 지원. `tossctl transactions overview --market kr|us`는 주문가능/출금가능/예정입금 요약. table/JSON/CSV 출력 지원 (Toss 200일 단일쿼리 캡 반영). 기여: @skyisle (PR #20)
- **영속(Persistent) 세션 캡처** — `auth login` 이 폰의 "이 기기 로그인 유지" 2차 인증까지 기다린 뒤 storage state 를 저장. 2차 인증 완료 시 Toss가 장기 SESSION 쿠키를 발급하므로 서버 idle timeout 면제 (≈1시간 후에도 401 안 남). `auth status` / `auth import-playwright-state` 출력에 `Persistence` 필드 추가 (`persistent (expires ...)` 또는 `session-scoped (≈1h idle timeout)`). JSON 출력에 `expires_at`, `persistent` 필드. 기여: @skyisle (PR #23)
- **원격/헤드리스 로그인** — `tossctl auth login --headless`. QR 탭 자동 활성화 + `/api/v2/login/wts/toss/{qr,status}` 응답 인터셉트로 QR URL과 확인 문자(answerLetter)를 stderr 출력. 텔레그램 등으로 URL만 폰에 보내 탭하면 Toss 앱이 열려 카메라 없이 인증. PNG 파일 저장은 `--qr-output <path>` (0600 권한). 기여: @skyisle (PR #22, 보안 강화 후 merge)
- **uv-managed Python 우선 탐지** — `auth login`이 helper Python을 찾을 때 `$TOSSCTL_AUTH_HELPER_PYTHON` → uv tool 관리 Python (`$UV_TOOL_DIR`, `$XDG_DATA_HOME/uv/tools`, `~/.local/share/uv/tools`, Windows `%APPDATA%/uv/tools`) → PATH의 `python3` 순서로 선택. `uv tool install ./auth-helper`로 전역 Python 오염 없이 helper 실행 가능. 기여: @keenranger (PR #21)

### Fixed
- **1시간 뒤 401 재발** — 과거 `auth login` 이 QR 1차 인증 직후 종료하여 session-scoped SESSION 만 저장했고, 약 1시간 idle 후 서버가 세션을 invalidate 하던 문제. "이 기기 로그인 유지" 2차 확인까지 기다려 persistent SESSION 저장하도록 변경. (참고: `docs/reverse-engineering/auth-notes.md` — Session Lifetime 섹션)

### Security
- 헤드리스 로그인의 `--qr-output` 파일을 `0o600` 권한으로 배타 쓰기 — 공유 머신에서 다른 사용자가 PNG 읽고 먼저 로그인 탭을 완료하는 시나리오 차단
- QR 응답 인터셉트가 path뿐 아니라 host(`wts-api.tossinvest.com`)까지 검증 — 동일 path suffix의 타 origin 응답 파싱 방지

## [0.3.6] - 2026-04-17

### Fixed
- **auth login 무한 대기 해결** — Python helper가 `DEVICE_INFO` localStorage 키를 필수로 기다리던 체크 제거. 토스 웹이 해당 키를 더 이상 보장하지 않아 로그인 성공 감지가 실패하던 회귀 수정 (Fixes #17, thanks to @pinion05)
- **`wts-api` 403 차단 해결** — `applySession`에 브라우저형 기본 `User-Agent` 설정. 기본 Go HTTP User-Agent(`Go-http-client/1.1`)가 토스 서버에서 핑거프린팅으로 차단되어 `account/*`, `portfolio/*`, `quote/*` 호출이 403을 받던 문제 해결. `auth login` 직후/10분 후 모두 정상 동작 확인 (Fixes #15, #17)

### Notes
- 명시적으로 `User-Agent`가 설정된 요청은 override되지 않고 그대로 유지됨

## [0.3.5] - 2026-03-30

### Added
- **테이블 출력 개선** — `portfolio positions`, `orders list`, `watchlist`, `quotes` 명령의 table 출력을 정렬된 컬럼 형식으로 변경. 종목명 좌측 정렬, 숫자 우측 정렬, 천단위 쉼표 적용
- `CONTRIBUTING.md` 추가

## [0.3.4] - 2026-03-28

### Fixed
- **auth login 브라우저 차단 해결** — Playwright 번들 Chromium 대신 시스템 Google Chrome 사용 (`channel="chrome"`). 토스증권이 `Sec-Ch-Ua` 헤더에서 `"Google Chrome"` 브랜드를 확인하도록 변경되어 Chromium이 차단됨 (Fixes #13)

### Changed
- `tossctl doctor` 브라우저 체크가 Chromium 대신 Chrome 감지로 변경
- `playwright install chromium` 불필요 — 시스템에 Google Chrome만 설치되어 있으면 됨

## [0.3.3] - 2026-03-24

### Added
- **USD 표시** — US 포지션에 매입가/현재가/평가금/손익을 USD로 병기 (by @seilk, PR #11)
- **설치 스크립트** — `curl -fsSL .../install.sh | sh` 한 줄 설치 (macOS/Linux)
- Issue/PR 템플릿, GitHub Sponsors 지원

### Fixed
- install.sh가 auth-helper를 누락하여 Linux에서 `auth login` 실패하던 문제 (Fixes #12)

## [0.3.2] - 2026-03-23

### Added
- **Cross-platform release builds** — Windows (amd64), Linux (amd64/arm64) 바이너리 자동 빌드
- Quick Start에 Windows/Linux 설치 가이드 추가

### Changed
- README Quick Start를 macOS/Linux/Windows 플랫폼별로 재구성
- 설치 섹션 중복 제거, Quick Start로 통합

### Docs
- architecture.md 갭 목록 업데이트 — sell, KR, fractional 구현 완료 반영
- README disclaimer 강화 (TOS 위반 가능성 명시)

## [0.3.1] - 2026-03-21

### Fixed
- US stock price rounding: `round4` → `round2` — prices now round to $0.01 (cent) precision instead of $0.0001, fixing `invalid.limit.price.scale` API errors
- `placeIntentSupported()` now accepts USD currency mode for fractional orders

### Changed
- README rewritten — restructured around feature tables, added fractional/KR examples, removed outdated sections, cleaner config reference

## [0.3.0] - 2026-03-21

### Added
- **Fractional (소수점) order support** — `tossctl order place --symbol TSLL --fractional --amount 18000`
  - US market only, market orders (시장가), amount-based (금액 기반)
  - `trading.fractional` config toggle (default: false)
  - `--amount` flag for specifying KRW amount
  - `--fractional` flag auto-selects market order type
- Fractional policy gate in `Place()` with "disabled by config" error
- `buildPlaceBody` fractional branch: `price=0, quantity=0, orderAmount=<KRW>, orderPriceType=01, isFractionalOrder=true`
- `placeIntentSupported()` now accepts fractional orders (US + market only)
- `NormalizePlace` validates fractional constraints (US only, amount required, auto market order)
- 10 new tests: fractional capability, policy, preview, payload, orderintent validation
- API compatibility verified via prepare dry-run (422 = payload accepted, insufficient balance)

## [0.2.3] - 2026-03-21

### Removed
- **MCP server** (`tossctl-mcp`) — CLI 자체가 AI 에이전트에서 직접 실행 가능하므로 불필요한 추상화 제거
- `make build-mcp` Makefile 타겟
- Release workflow에서 tossctl-mcp 바이너리

## [0.2.2] - 2026-03-21

### Added
- `tossctl quote batch <symbol> [symbol...]` — fetch multiple stock quotes at once
- `tossctl export positions --market kr|us|all` — filter exported positions by market
- `tossctl export orders --market kr|us|all` — filter exported orders by market
- Quote output tests (6 test cases)

### Fixed
- Floating point display artifacts in quote batch table output (e.g., `-0.8500000000000014` → `-0.85`)

## [0.2.1] - 2026-03-21

### Added
- MCP server unit tests (10 test cases) covering initialize, tools/list, tool calls, error handling
- Refactored MCP server to testable pure functions (handleMethod, buildInitializeResponse, buildToolsList)

### Removed
- Unused `stub.go` command helper (export commands now fully implemented)

## [0.2.0] - 2026-03-21

### Added
- **MCP server** (`tossctl-mcp`) — read-only Model Context Protocol server for AI agent integration
  - `get_portfolio_positions` — 보유 포지션 조회
  - `get_account_summary` — 계좌 요약 조회
  - `get_quote` — 종목 시세 조회 (US/KR)
  - `list_pending_orders` — 미체결 주문 조회
  - `list_completed_orders` — 체결 완료 내역 조회 (market filter 지원)
  - `list_watchlist` — 관심 종목 조회
- `tossctl export positions` — CSV 포지션 내보내기 (stub에서 실제 구현으로 전환)
- `tossctl export orders` — CSV 체결 내역 내보내기 (stub에서 실제 구현으로 전환)
- `make build-mcp` Makefile 타겟
- Release workflow에 `tossctl-mcp` 바이너리 포함

## [0.1.7] - 2026-03-21

### Added
- Korean stock (국내주식) trading support — `tossctl order place --symbol 005930 --market kr`
- `trading.kr` config toggle (default: false) — KR requires `trading.place` and `trading.kr`
- KR branch in `buildPlaceBody`: raw KRW price, no USD conversion, no `allowAutoExchange`
- KR branch in `PlacePendingOrder`: skip USD exchange rate fetch and FX consent
- `InferMarketFromStockCode()` for cancel/amend market recovery from stock code pattern
- KR symbol detection in `NormalizePlace`: numeric 6-digit + market=us → error with guidance
- 13 new test cases (T1-T13) for KR gate, preview, config, payload, reconciliation, symbol detection
- KR cancel/amend live verification TODO

### Changed
- `placeIntentSupported()` now allows both "us" and "kr" markets (was us-only)
- `Place()` reordered: capability → market policy → side policy → execution guard (was guard-first)
- Reconciliation functions parameterized: `Market: "us"` hardcoding (8 places) → market parameter
- TODOS.md: reconciliation parameterization marked as completed
- README, architecture.md, configuration.md updated with KR documentation

## [0.1.6] - 2026-03-21

### Added
- Sell order support for US limit / KRW / non-fractional — `tossctl order place --side sell`
- `trading.sell` config toggle (default: false) — sell requires both `trading.place` and `trading.sell` to be enabled
- Sell policy gate in `Place()` with distinct "disabled by config" error (not "unsupported")
- Sell state reflected in `PreviewPlace` warnings and `MutationReady`
- Sell toggle visible in `config show`, `doctor`, and `EnabledActions()`
- 10 new test cases (T1-T10) covering sell gate, preview, config parsing, payload, and error messages
- `TODOS.md` for tracking deferred work (reconciliation parameterization, sell live verification)

### Changed
- `placeIntentSupported()` no longer restricts by side — both buy and sell are broker-capable
- Warning message updated: "US buy/sell limit orders" (was "US buy limit orders")
- `ErrPlaceUnsupported` message no longer includes `--side buy`
- Doctor trading_scope check updated to reflect buy/sell support
- README, architecture.md, configuration.md updated with sell documentation
