# Changelog

All notable changes to this project will be documented in this file.

## [0.4.13] - 2026-05-14

### Added
- **자동 업데이트 알림** — 명령 실행 후 (성공 시) GitHub Releases 의 latest stable tag 를 조회해 새 버전이 있으면 stderr 에 한 줄 안내. 24h 디스크 캐시 (`<cache>/tossctl/update-check.json`), 네트워크 실패는 silent (다음 실행 때 재시도). config `update_check.enabled` (기본 `true`) 로 끌 수 있음. JSON/CSV 출력, non-tty (cron 등), dev 빌드, prerelease tag 에서는 자동 skip — 자동화 출력은 절대 오염시키지 않음.
- `internal/updatecheck` 패키지 + 단위 테스트 (semver 비교, 캐시 hit/miss, 네트워크 실패 시 stale fallback).

### Changed
- `config show` / `doctor` 의 status 출력에 `update_check` 표시 (schema_version 은 그대로 2 — 신규 필드는 additive 라 마이그레이션 불필요).

## [0.4.12] - 2026-05-14

### Fixed
- **Windows 에서 `tossctl auth login` 이 `AttributeError: module 'os' has no attribute 'fchmod'` 로 끊기던 회귀 (#26)** — `os.fchmod` 는 Unix 전용이라 Windows Python 에서는 노출되지 않음. auth-helper 의 권한 설정을 `_set_private_permissions` 헬퍼로 통일하고, fchmod 가 없으면 chmod 로 fallback. chmod 의 fd 인자 지원도 platform-dependent 라 (`os.supports_fd`), 일부 Windows 환경에서 `TypeError`/`OSError` 가 발생하면 best-effort 로 silently skip — 권한 설정은 못 해도 storage-state 파일 저장은 정상 진행. 회귀 검증 단위 테스트 3개 추가. (제보 + 수정: @netics01)

## [0.4.11] - 2026-05-13

### Changed
- `monitor` 모듈 잔여 over-abstraction 정리. `expectStatus` 의 사용 안 되는 `body` 파라미터 제거 (호출자 6곳 정리), `printResults` 의 인라인 익명 인터페이스를 `io.Writer` 로 치환, `monitor api --help` 의 Discord-specific 예시 제거 (alert recipe 는 `AGENTS.md` 한 곳으로 단일화). `Probe` 필드 인라인 주석 정리. 동작 변화 없음 — 코드만 unix 철학에 맞게 더 좁힘.

## [0.4.10] - 2026-05-13

### Changed
- `monitor api` 시그니처 단순화: `--webhook` / `TOSSCTL_MONITOR_WEBHOOK` 제거. exit 0/1 만 반환하고 알림 채널 (Discord · Slack · ntfy · macOS notification · 이메일 등) 은 cron 라인의 `|| <command>` 우항에서 사용자가 합성. 합성 recipe 는 신규 `AGENTS.md`.

### Removed
- `internal/monitor/discord.go` (Discord webhook 송출 헬퍼). 동등한 효과는 한 줄 `curl` 합성으로 충분.

### Added
- `AGENTS.md` — OpenClaw · Claude Code · Codex · Cursor 등 AI 에이전트가 `monitor api` 와 알림 채널을 cron 으로 묶을 때 참고할 짧은 recipe 모음.

## [0.4.9] - 2026-05-13

### Added
- **`tossctl monitor api`** — 6개 read-only endpoint (account-list, summary-overview, positions, watchlist, quote, pending-orders) 에 schema-invariant probe 실행. 핵심 JSON 경로/타입 검증이 실패하면 exit 1 + 선택적 Discord webhook 알림 (`--webhook` 또는 `TOSSCTL_MONITOR_WEBHOOK`). 본인 세션으로 본인 머신에서 본인이 설정한 webhook 한 곳에만 보고. #29 같은 토스 서버측 body 계약 변경을 cron 으로 조기 감지하기 위한 도구. 설정 가이드: `docs/operations.md`.

### Changed
- `expectStatus` 가 webhook 알림에 status code + 기대값만 노출 (응답 본문 fragment 미포함). 단위 테스트 `TestProbeChecksDoNotLeakResponseBodyOnFailure` 로 회귀 방지.

## [0.4.8] - 2026-05-13

### Fixed
- **`portfolio positions` · `watchlist list` 회귀 (#29)** — 2026-05-13 16:10 KST 즈음 토스 서버가 `/api/v2/dashboard/asset/sections/all` 의 body 계약을 변경. 기존 빈 `{}` body는 여전히 200을 반환하지만 sections가 빈 배열 + `pollIntervalMillis: 3000` 만 내려와, CLI가 "SORTED_OVERVIEW section not found" / "WATCHLIST section not found"로 실패했음. 새 계약은 `{"types":[<section-name>]}` 필터 필수. 두 커맨드 모두 해당 필터를 송출하도록 수정. (제보: kwakmu18, #29)
- 라이브 캡처 결과 `docs/reverse-engineering/rpc-catalog.md` 에 body 형식 명시.

### Notes
- 다른 read-only API (`account`, `orders`, `quote`, `transactions`, `account summary` 등)는 이번 토스 변경에 영향 없음이 같은 세션으로 확인됨.

## [0.4.7] - 2026-05-06

### Fixed
- **`order place --fractional --currency-mode USD` 거부 (#28)** — 와이어 페이로드는 항상 `currencyMode="KRW"`인데 (토스 스펙) Fractional 분기는 `intent.Amount`를 USD/KRW 구분 없이 그대로 `orderAmount`에 실어 보내고 있었음. 결과적으로 `--amount 100 --currency-mode USD`가 서버에 ₩100으로 전달되어 "금액주문은 $1 또는 1,000원 이상" 오류가 났음. 이제 USD 모드일 때는 `intent.Amount * meta.ExchangeRate`로 KRW 변환 후 송출. `--currency-mode KRW`(기본) 동작은 그대로. (제보: @leesj10147)

## [0.4.6] - 2026-05-06

### Added
- **`tossctl auth extend`** — 폰 토스 앱 푸시 승인을 통해 서버 측 세션 만료를 연장합니다. 블로킹 + Braille 스피너 UX, `--timeout` (기본 120초), `Ctrl+C` 처리. 사용 endpoint chain: `POST /api/v1/wts-login-extend/doc/request` → polling `GET /doc/{txId}/status` (`REQUESTED` → `COMPLETED`) → `POST /api/v1/wts-login-extend/{txId}/state` (필수 finalize) → `GET /api/v1/session/expired-at`. 약 7일 연장됩니다. 기여: @skyisle (PR #27)
- **24h 만료 경고** — 모든 명령 시작 시 서버 측 세션 만료가 24시간 미만으로 남았으면 stderr 한 줄로 경고 (`⚠ session expires in ~21h 58m; run \`tossctl auth extend\` to renew`). `--output json` 모드와 `auth/version/help` skip-list에서는 silent.
- **`session.json` 에 `server_expires_at` 필드 추가** — 서버 측 ~7일 활성 만료 시계 (기존 `expires_at` 의 1년 쿠키 만료와 별개). `auth status` table에 `Server Expiry: YYYY-MM-DD HH:MM KST` 줄 추가, JSON 출력에 `server_expires_at`.

### Changed
- `ExtensionTimeoutError{Elapsed}` 타입 도입 — 기존 `fmt.Errorf("%w (waited %s)", ...)` + caller-side `extractParenDetail` 문자열 파싱을 typed wrapper로 대체. 에러 메시지 포맷이 바뀌어도 elapsed 추출이 안 깨짐.
- `formatKST(t)` helper 추출 — `auth status` / `auth extend` 출력의 `2006-01-02 15:04 KST` 포맷이 한 곳에서 관리됨.

### Fixed
- v0.4.0의 persistent SESSION 쿠키(1년)는 정상이어도 토스 서버 측 ~7일 활성 timer가 별도로 흐르는 사실이 그동안 묵묵히 401을 일으키고 있었음. `auth extend`로 매번 `auth login`(QR + 폰 풀 플로우)을 다시 돌지 않고 폰 푸시 한 번으로 연장 가능. (참고: `docs/reverse-engineering/auth-notes.md` Session Lifetime 섹션)

## [0.4.5] - 2026-04-29

### Added
- **`tossctl push listen` — SSE 푸시 리스너** — 토스증권 웹이 `GET https://sse-message.tossinvest.com/api/v1/wts-notification` 로 제공하는 Server-Sent Events 채널 구독. 세션 쿠키 재사용, JSONL 로 stdout 출력, 기본 exponential backoff 재연결 (`--retry=false` 로 비활성 가능), 토스 서버의 graceful `event: connection-close` 핸드오프 즉시 재연결. Toss 는 thin notification 방식이라 payload 대신 `{"type":"pending-order-refresh",...}` 같은 "재조회하라" 신호만 내려줌 (관찰된 이벤트 타입: `pending-order-refresh`, `purchase-price-refresh`, `share-holdings`, `web-push` 등). 구독 가능한 이벤트 타입과 의미는 `docs/reverse-engineering/push-events.md` 에 정리. 기여: @skyisle (PR #25)

### Changed
- **공통 User-Agent 상수 통합** — `client.defaultBrowserUserAgent` (private) 를 `client.DefaultBrowserUserAgent` (export) 로 승격하고 `internal/push` 도 동일 상수 재사용. 기존엔 두 패키지에 동일 Chrome UA 문자열이 중복돼 있어 향후 Chrome 버전 bump 시 두 곳을 동기화해야 했음 — 이제 한 군데만 수정하면 `wts-api`/`wts-cert-api`/`wts-info-api`/`sse-message` 전 채널 fingerprint가 같이 갱신됨.

## [0.4.4] - 2026-04-23

### Added
- **US 지정가 주문에서 USD 가격 입력 허용** — `order place --currency-mode USD --price 158.01 ...`. 기존엔 비 fractional 경로가 `CurrencyMode=KRW`만 허용해 `Live Ready=false`로 떨어졌음. 입력만 USD로 받고 와이어 페이로드는 기존과 동일하게 `currencyMode="KRW"`+USD 가격 필드 (캡처된 토스 웹 UI 스펙과 일치). `--currency-mode KRW`(기본)는 환율 변환 경로 그대로 유지. 기여: @skyisle (PR #24)

### Changed
- `buildPlaceBody`의 USD branch 가드를 `case intent.Market == "us" && intent.CurrencyMode == "USD"`로 명시화 — 향후 market 추가 시 암묵적 fall-through 방지 (현재 `placeIntentSupported` 화이트리스트로 안전하지만 defense-in-depth)

## [0.4.3] - 2026-04-23

### Removed
- **죽은 config 토글 3개 제거** — 모두 코드 어디서도 참조되지 않아 UX 오해만 유발하던 필드. 기존 config에 남아있어도 무시되며 `doctor`의 `legacy_config`에서 감지됨:
  - `trading.grant` — `order permissions grant` 커맨드 게이트처럼 보였지만 실제 매매 게이트는 `place`/`cancel`/`amend` + `allow_live_order_actions`만 확인. `grant=false, place=true` 설정 시 매매는 되면서 grant만 막히는 기묘한 조합 가능하던 문제
  - `dangerous_automation.complete_trade_auth` — 어떤 handler에도 연결되지 않은 dead code. `true`로 켜면 doctor에서 경고만 뜨고 기능은 무변화
  - `dangerous_automation.accept_product_ack` — 동일 (dead code)

### Changed
- `order permissions grant` 실행 조건 단순화 — `place` 또는 `cancel` 또는 `amend` 중 하나라도 허용되어 있으면 실행 가능 (기존: 별도의 `grant` 토글 추가 요구)
- README / `docs/configuration.md` 토글 섹션 재구성 — "경로 게이트" (`place`/`cancel`/`amend`, broker API 분기) vs "스코프 선언" (`sell`/`kr`/`fractional`, 유저 자가 제한) 두 범주로 명확히 구분해서 설명. 기존에 "안전 게이트"로 묶여있어 오해 소지 있던 부분 정리
- `config.schema.json` 업데이트 — 제거된 필드들 `required`에서 빠짐

### Migration
- 별도 조치 불필요. v0.4.2에서 오던 config를 그대로 두면 됨. `doctor` 돌리면 `legacy_config` info에 "trading.grant / complete_trade_auth / accept_product_ack는 v0.4.3부터 무시됨"이 표시됨
- 혹시 `trading.grant=true`에만 의존하고 실제 매매 토글(`place` 등)은 꺼둔 상태였다면, 동작상 차이 없음 — 예전에도 실제 매매는 불가능했음

## [0.4.2] - 2026-04-23

### Added
- **`tossctl doctor --report`** — JSON 진단 번들. 기존 doctor 정보 + `wts-api` · `wts-cert-api` · `wts-info-api` 각 1회 실시간 probe (200/401/403 + 응답시간) + 파일 권한 audit (`session.json`/`config.json`/... 가 0600, 디렉토리 0700인지) + orphan intermediate 파일 탐지를 한 덩어리로 출력. 홈 디렉토리 경로는 `~`로 자동 redact되어 사용자명이 노출되지 않으므로 GitHub 이슈에 그대로 첨부 가능.
- Issue 템플릿 `bug_report.yml`에 `tossctl doctor --report` 필드 추가 — 제보자가 실행 결과를 JSON으로 붙이면 유지보수자가 한눈에 환경·세션·엔드포인트 상태를 파악할 수 있음. 기존 `--report` 없이는 각 이슈마다 "어떤 endpoint가 403?" 을 개별로 물어봐야 했던 오버헤드 제거.

### Notes
- v0.4.1 이전에 생성된 tossctl 디렉토리(`~/Library/Application Support/tossctl/` 등)는 여전히 `0755` 모드로 남아있을 수 있음. `doctor --report`의 `file_modes` 항목에서 `expected: "0700"` 과 비교해 `ok: false`로 표시되며, 원하는 사용자는 `chmod 0700 <dir>` 로 수동 정리 가능. 기능에 영향 없음.

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
