# Push Events (SSE)

Verified 2026-04-23 from the authenticated dashboard with the tossctl session.

## Channel

Toss Securities web does **not** use WebSocket. Server → client push is delivered via Server-Sent Events (SSE) on a single long-lived HTTP GET.

- Endpoint: `GET https://sse-message.tossinvest.com/api/v1/wts-notification`
- Required headers: `Accept: text/event-stream`, `Cache-Control: no-cache`
- Auth: standard session cookies (`SESSION`, `XSRF-TOKEN`, `UTK`, `LTK`, `FTK`, `BTK`, `browserSessionId`, `deviceId`) — same set used for `wts-api` calls
- Response: `Content-Type: text/event-stream; charset=UTF-8`, chunked

A second push channel exists for OS-level Web Push (VAPID) via `GET https://wts-cert-api.tossinvest.com/api/v1/personalize/wts/browser-push/vapid` + a subscribe POST. Out of scope for the in-CLI listener.

## Stream Shape

On connect the server emits a `retry: 3600000` directive, then alternates SSE comment heartbeats (`:heartbeat`) with event frames. The server seems to re-emit the `retry` directive roughly every 2 seconds alongside heartbeats.

Frames follow the standard SSE format:

```
id: <32-hex-char opaque id>
data: <JSON payload>

```

`data` is always a single-line JSON object with at minimum `type` and `key`.

## Observed Event Types

Triggered by a single `order cancel` on a pending US order (MRVL, order id `2026-04-23/5`):

| Event count | `data.type` | Other `data` fields |
| --- | --- | --- |
| 3 | `pending-order-refresh` | `key` (string, observed `"1"`) |
| 3 | `purchase-price-refresh` | `msg.stockCode`, `key` |

Example frames (timestamps from the capture run):

```
[18:57:38] {"id":"e8b4085b71ed4c5e8b2f16539cc7d8fd", "data":"{\"type\":\"pending-order-refresh\",\"key\":\"1\"}"}
[18:57:38] {"id":"be147d920c45474381207babdc45e27d", "data":"{\"type\":\"purchase-price-refresh\",\"msg\":{\"stockCode\":\"US20000627001\"},\"key\":\"1\"}"}
```

The event count for a single cancel (prepare → cancel → reconcile) is 3 per type. Expect the same multiplicative pattern for place and amend flows.

## Semantics

Toss uses SSE as a **thin notification** channel — events signal "something changed, re-fetch the relevant resource" rather than carrying the payload. Consumers must pair each event type with a follow-up REST call to get the actual state.

Suggested mapping for the CLI:

| Event type | Re-fetch |
| --- | --- |
| `pending-order-refresh` | `/api/v1/trading/orders/histories/all/pending` |
| `purchase-price-refresh` | `/api/v1/trading/orders/calculate/<stockCode>/average-price` + orderable-amount / cost-basis endpoints as needed |

Unknown event types seen in the wild should be logged verbatim so the taxonomy can grow.

## Known Unknowns

- Event types triggered by `order place` and `order amend` — not yet captured on this channel.
- Fill notifications during market hours — the cancel capture happened pre-open (US market closed), so fill-related events are not yet observed.
- Long-idle behavior of the connection — Toss's `retry: 3600000` (1 hour) suggests they expect EventSource to auto-reconnect; the SESSION cookie is persistent and should survive reconnects, but token refresh behavior is not exercised.
- Payload size ceiling — no frame larger than ~200 bytes has been observed.
