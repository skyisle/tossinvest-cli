# Auth Notes

Verified from public page behavior on 2026-03-11. Last updated 2026-04-21 (persistent SESSION via "이 기기 로그인 유지" capture).

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

## Session Lifetime

Two distinct SESSION cookie shapes exist and their idle-timeout behavior differs:

| SESSION kind | `Set-Cookie` attributes | Server idle timeout | Triggered by |
|---|---|---|---|
| session-scoped (default after QR) | `SESSION=...; Secure; HttpOnly; SameSite=Strict` (no `Max-Age`) | **≈1 hour sliding** — every authenticated call resets to `now + 1h` | `POST /api/v3/login/ticket` after QR auth step 1 |
| persistent (long-lived) | `SESSION=...; Max-Age=31536000; Expires=<1 year ahead>; HttpOnly` | **Exempt** — survives multi-hour idle (verified with 60+ min idle probe, 200 OK after gap) | `POST /api/v1/wts-login-device/check-with-login` **after** user confirms "이 기기 로그인 유지" on phone |

Consequences:
- If the CLI captures storage state right after QR step 1 only, the saved SESSION is session-scoped and the CLI will 401 after ≈1h of inactivity.
- To obtain the persistent SESSION the QR flow has a **second** confirmation step on the Toss app ("이 기기 로그인 유지"). The Python auth-helper waits for this before saving (`has_persistent_session_cookie` — SESSION `expires` > 1 week out).
- `GET /api/v1/session/expired-at` returns the current session expiry as KST RFC3339. It also bumps the server-side idle timer (we could not isolate it as a pure read in tests).
- No "silent re-auth via long-lived token" exists. Without a valid SESSION the server's `/api/v3/init` issues a fresh guest SESSION and the page redirects to `/signin`. `UTK/LTK/FTK/BTK` alone do not prove identity.
- Browser tabs appear to "persist for days" only because (a) the dashboard polls authenticated APIs every 3–30 seconds while the tab is open (≈460 calls observed over 18 min of user idle), keeping the idle timer warm, or (b) the browser has a persistent SESSION from having confirmed "이 기기 로그인 유지".

## Logout Behavior (Set-Cookie on 2xx)

Several `wts-cert-api.tossinvest.com` endpoints respond to 2xx requests with `Set-Cookie: SESSION=; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Secure; HttpOnly; SameSite=Lax` — an instruction to **delete** the client-side SESSION cookie. Observed at least on:

- `GET /api/v1/dashboard/common/cached-orderable-amount`
- `POST /api/v1/dashboard/asset/sections/all`
- `POST /api/v2/dashboard/asset/sections/all`
- `GET /api/v1/properties/member/shared`

Browsers and the tossctl CLI both **ignore** this (CLI has no cookie jar; browsers apparently don't honor the delete either, because their existing SESSION has `SameSite=Strict` while the delete instruction carries `SameSite=Lax`, producing a cookie-jar mismatch). If the CLI ever adopts an `http.CookieJar` naively, it will self-destruct its session on these responses. The safe rule is: **never apply `SESSION=; Max-Age=0` delete instructions**.

## HTTP Client Fingerprinting

토스 서버는 `wts-api`, `wts-cert-api`, `wts-info-api` 모두 기본 Go HTTP User-Agent(`Go-http-client/1.1`)를 403으로 거부한다. v0.3.6부터 `applySession`이 브라우저형 기본 `User-Agent`(Chrome)를 설정하며, 호출자가 명시적으로 `User-Agent`를 세팅한 경우에는 override하지 않는다.

## Guardrails

- do not store raw login captures in git
- do not commit cookies or personally identifying information
- do not implement trading-related flows in the auth helper
