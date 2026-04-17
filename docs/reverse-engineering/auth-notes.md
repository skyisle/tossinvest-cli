# Auth Notes

Verified from public page behavior on 2026-03-11. Last updated 2026-04-17 (v0.3.6 — `DEVICE_INFO` 요건 제거, HTTP 기본 UA 갱신).

## Public Behavior

- Navigating to `https://www.tossinvest.com/account` without an authenticated session redirected to `https://www.tossinvest.com/signin?redirectUrl=%2Faccount`.
- The sign-in page exposed two visible entry modes:
  - phone-based login
  - QR-code login
- Public network activity on the sign-in page included `POST https://wts-api.tossinvest.com/api/v2/login/wts/toss/cert-init`.

## Authenticated Browser Observations

Captured from a real browser session on 2026-03-11 after QR login.

Observed login flow endpoints:

- `POST /api/v2/login/wts/toss/cert-init`
- `POST /api/v2/login/wts/toss/qr`
- repeated `GET /api/v2/login/wts/toss/status`
- `POST /api/v2/login/wts/toss`
- `POST /api/v3/login/ticket`

Observed cookie names in browser storage state:

- `deviceId`
- `browserSessionId`
- `XSRF-TOKEN`
- `SESSION`
- `UTK`
- `LTK`
- `FTK`
- `BTK`

Observed local storage keys:

- `WTS-DEVICE-ID`
- `qr-tabId`
- `WTS-SYNC-SEED`
- `login-method`
- `DEVICE_INFO` — v0.3.6 기준 더 이상 항상 생성되지 않음. 로그인 성공 판정은 `WTS-DEVICE-ID` + `login-method`만 요구

Observed session storage keys:

- `WTS-BROWSER-TAB-ID`

These observations suggest that both cookies and browser storage values matter for a durable WTS session. The auth helper should capture both, not cookies alone.

## Working Assumption

The CLI should not attempt to recreate the login flow in Go.

Instead:

- a Python auth helper opens a real browser
- the user completes login manually
- the helper extracts the minimum session state needed for subsequent read-only HTTP calls
- the Go CLI stores and reuses that state

## Unknowns To Capture

- which subset of cookies are strictly required after successful login
- ~~which subset of local storage or session storage values are strictly required~~ — v0.3.6에서 `WTS-DEVICE-ID` + `login-method`로 확정
- whether request signing or CSRF tokens are needed for authenticated read endpoints
- whether session state differs between phone and QR flows
- whether the web app refreshes sessions silently

## HTTP Client Fingerprinting

토스 서버는 `wts-api`, `wts-cert-api`, `wts-info-api` 모두 기본 Go HTTP User-Agent(`Go-http-client/1.1`)를 403으로 거부한다. v0.3.6부터 `applySession`이 브라우저형 기본 `User-Agent`(Chrome)를 설정하며, 호출자가 명시적으로 `User-Agent`를 세팅한 경우에는 override하지 않는다.

## Guardrails

- do not store raw login captures in git
- do not commit cookies or personally identifying information
- do not implement trading-related flows in the auth helper
