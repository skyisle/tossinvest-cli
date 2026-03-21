# Architecture

`tossinvest-cli`는 토스증권 웹 세션을 재사용하는 비공식 CLI입니다. 현재 구조는 `브라우저 로그인 보조 도구 + Go CLI + 로컬 상태 파일 + Toss 증권 웹 API`로 나뉘어 있습니다.

이 문서는 현재 저장소에 구현된 기준으로 아키텍처를 정리합니다. 목표는 두 가지입니다.

- 사람이 전체 구조를 빠르게 이해할 수 있게 하기
- 이후 LLM이나 다른 자동화 도구가 같은 경계를 기준으로 기능을 확장할 수 있게 하기

## 현재 상태

현재 구현 범위는 아래와 같습니다.

- 브라우저 로그인 기반 세션 저장과 재사용
- 계좌, 포트폴리오, 미체결 주문, 관심종목, 시세 조회
- `orders completed`, `order show <id>` 기반 주문 상태 조회
- 거래 베타
  - `US/KR buy/sell limit` + `US fractional (market)`
  - sell: `trading.sell=true`, KR: `trading.kr=true`, fractional: `trading.fractional=true`
  - `KRW`
  - `place`
  - same-day pending `cancel`
  - `amend` wiring 존재, 추가 live 검증 필요

거래는 기본적으로 꺼져 있습니다. 사용자가 `config.json`에서 기능별로 직접 열고, 그 다음에도 permission grant와 explicit flags를 통과해야만 mutation이 실행됩니다.

## 설계 원칙

- 기본은 `read-only`
- 로그인 획득과 API 호출을 분리
- 사람과 에이전트 모두 같은 CLI 표면을 사용
- 거래 mutation은 별도 gate와 reconciliation을 거침
- 상태는 로컬 파일로 명시적으로 저장
- discovery 문서와 fixture는 코드와 분리해서 유지

## System Context

```mermaid
C4Context
title System Context - tossinvest-cli

Person(user, "User", "터미널에서 직접 조회와 거래를 실행")
Person(agent, "Agent / Automation", "Claude, Codex, shell script, OpenClaw 같은 상위 자동화 계층")

System(cli, "tossinvest-cli", "토스증권 웹 세션을 재사용하는 비공식 CLI")

System_Ext(tossWeb, "Toss Securities Web", "브라우저 로그인과 웹 UI")
System_Ext(tossApi, "Toss Securities Web APIs", "조회와 거래 요청 대상")
System_Ext(playwright, "Playwright Chromium", "실제 브라우저 로그인 세션 확보")

Rel(user, cli, "실행", "CLI")
Rel(agent, cli, "호출", "CLI / JSON")
Rel(cli, playwright, "로그인 시 실행", "Python helper")
Rel(playwright, tossWeb, "브라우저 로그인", "HTTPS")
Rel(cli, tossApi, "조회 / 거래 요청", "HTTPS + session cookies")
Rel(playwright, cli, "storage state 전달", "JSON file")
```

## Container View

```mermaid
C4Container
title Container Diagram - tossinvest-cli

Person(user, "User", "직접 사용하는 사람")
Person(agent, "Agent", "LLM 또는 스크립트")

System_Boundary(system, "tossinvest-cli") {
  Container(goCli, "tossctl", "Go + Cobra", "메인 CLI 바이너리")
  Container(authHelper, "auth-helper", "Python + Playwright", "브라우저 로그인 보조 도구")
  ContainerDb(localFiles, "Local State Files", "JSON files", "config.json, session.json, trading-permission.json")
  Container(docs, "Tracked Docs + Fixtures", "Markdown + JSON", "reverse engineering, trading notes, sanitized fixtures")
}

System_Ext(tossWeb, "Toss Securities Web", "브라우저 로그인 화면")
System_Ext(tossApi, "Toss Securities Web APIs", "조회 / 거래 엔드포인트")

Rel(user, goCli, "Uses")
Rel(agent, goCli, "Calls", "JSON / shell")
Rel(goCli, authHelper, "Starts for login", "subprocess")
Rel(authHelper, tossWeb, "Captures login session", "browser automation")
Rel(authHelper, localFiles, "Writes storage state-derived session", "JSON")
Rel(goCli, localFiles, "Reads/writes config, session, grant", "JSON files")
Rel(goCli, tossApi, "Calls read-only and trading APIs", "HTTPS")
Rel(goCli, docs, "Uses captured knowledge", "dev workflow")
```

## Component View

```mermaid
C4Component
title Component Diagram - tossctl

Container(cli, "tossctl", "Go CLI", "Main binary")

Container_Boundary(core, "Application Core") {
  Component(commands, "cmd/tossctl", "Cobra commands", "Command entrypoints and flag parsing")
  Component(configSvc, "internal/config", "Config service", "Loads config.json and schema-aware defaults")
  Component(authSvc, "internal/auth", "Auth service", "Runs auth-helper and imports browser session")
  Component(doctorSvc, "internal/doctor", "Doctor service", "Checks local environment, config, session, helper readiness")
  Component(clientSvc, "internal/client", "HTTP client", "Calls Toss web APIs and parses responses")
  Component(tradingSvc, "internal/trading", "Trading service", "Preview, gate, mutation orchestration, reconciliation")
  Component(intentSvc, "internal/orderintent", "Intent normalization", "Canonical order inputs and confirm tokens")
  Component(permissionSvc, "internal/permissions", "Grant service", "Short-lived local trading grant")
  Component(sessionSvc, "internal/session", "Session store", "Persists imported browser session")
  Component(outputSvc, "internal/output", "Renderers", "Table / JSON / CSV output")
}

Rel(commands, configSvc, "Loads config")
Rel(commands, authSvc, "Runs login/status/logout")
Rel(commands, doctorSvc, "Runs doctor")
Rel(commands, clientSvc, "Executes read-only commands")
Rel(commands, tradingSvc, "Executes preview/place/cancel/amend")
Rel(tradingSvc, intentSvc, "Uses canonical inputs")
Rel(tradingSvc, permissionSvc, "Requires grant")
Rel(tradingSvc, clientSvc, "Runs trading prepare/create/cancel/correct")
Rel(authSvc, sessionSvc, "Stores session")
Rel(clientSvc, sessionSvc, "Reads session")
Rel(commands, outputSvc, "Renders results")
```

## Key Flows

### 1. Browser-assisted login

```mermaid
sequenceDiagram
    actor User
    participant CLI as tossctl auth login
    participant Helper as auth-helper
    participant Browser as Playwright Chromium
    participant TossWeb as Toss Web
    participant Session as session.json

    User->>CLI: auth login
    CLI->>Helper: start login helper
    Helper->>Browser: launch browser
    Browser->>TossWeb: open login/account page
    User->>Browser: complete QR login
    Browser-->>Helper: cookies + localStorage
    Helper-->>CLI: storage-state path
    CLI->>Session: import session.json
```

### 2. Read-only command

```mermaid
sequenceDiagram
    actor Caller as User / Agent
    participant CLI as tossctl
    participant Config as config.json
    participant Session as session.json
    participant Client as internal/client
    participant API as Toss Web APIs

    Caller->>CLI: account summary / quote / orders list
    CLI->>Config: load defaults
    CLI->>Session: load session
    CLI->>Client: execute request
    Client->>API: GET/POST with stored session
    API-->>Client: response JSON
    Client-->>CLI: parsed domain model
    CLI-->>Caller: table/json/csv
```

### 3. Trading mutation with safety gates

```mermaid
sequenceDiagram
    actor Caller as User / Agent
    participant CLI as tossctl
    participant Config as config.json
    participant Grant as trading-permission.json
    participant Trading as internal/trading
    participant Client as internal/client
    participant API as Toss trading APIs

    Caller->>CLI: order preview
    CLI->>Trading: canonical preview + confirm token
    Trading-->>Caller: preview

    Caller->>CLI: order place/cancel/amend --execute --dangerously-skip-permissions --confirm
    CLI->>Config: verify action enabled
    CLI->>Grant: verify active grant
    CLI->>Trading: guard(action, flags, confirm)
    Trading->>Client: prepare/mutation call
    Client->>API: prepare
    API-->>Client: orderKey / challenge / reject
    Client->>API: final mutation
    API-->>Client: accepted
    Client->>API: pending/completed reconciliation
    Client-->>Trading: MutationResult
    Trading-->>Caller: accepted_pending / filled_completed / canceled / amended_pending / unknown
```

## Safety Model

거래 mutation은 아래 순서로 열립니다.

1. `config.json`
   - `grant`
   - `place`
   - `cancel`
   - `amend`
   - `allow_live_order_actions`
   - `dangerous_automation.*`
2. `order permissions grant --ttl ...`
3. `--execute`
4. `--dangerously-skip-permissions`
5. `--confirm <token>`

즉, config가 열려 있어도 매번 CLI 실행 시점의 명시적 확인이 필요합니다.

## Local State

로컬 상태는 아래 파일로 관리됩니다.

| File | Role |
|---|---|
| `config.json` | 거래 기능 허용 여부 |
| `session.json` | 브라우저에서 가져온 세션 |
| `trading-permission.json` | 짧은 TTL의 거래 grant |

기본 경로는 `tossctl doctor`와 `tossctl config show`로 확인할 수 있습니다.

## Package Map

| Package | Role |
|---|---|
| `cmd/tossctl` | 명령 진입점 |
| `internal/config` | config.json 로드, 기본값, schema |
| `internal/auth` | 로그인 orchestration, 세션 import |
| `auth-helper/` | Python + Playwright 로그인 보조 |
| `internal/client` | Toss 웹 API 호출과 응답 파싱 |
| `internal/trading` | preview, gate, mutation orchestration |
| `internal/orderintent` | canonical input, confirm token |
| `internal/permissions` | trading grant 관리 |
| `internal/session` | session.json 저장 |
| `internal/output` | table/json/csv 렌더링 |
| `docs/reverse-engineering` | read-only discovery 문서 |
| `docs/trading` | trading discovery 문서 |

## Current Gaps

아래는 아직 남아 있는 항목입니다.

- `amend`의 추가 live 검증
- `place`와 `amend`의 상태 판별 추가 검증
- `sell`, `market`, `KR`, `fractional` 확장
- interactive auth challenge가 필요한 mutation 분기 일반화

## Related Docs

- [`configuration.md`](./configuration.md)
- [`trading/README.md`](./trading/README.md)
- [`reverse-engineering/`](./reverse-engineering/)
