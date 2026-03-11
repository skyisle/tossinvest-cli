# Toss Investment CLI Trading Expansion Design

## Summary

This document extends the current read-only `tossctl` architecture to support order placement, cancelation, and amendment while preserving an intentionally high-friction safety model.

The CLI must remain safe by default. Real trading actions are allowed only after explicit preview, confirmation, and permission bypass steps. The design treats trading as a separate capability set from read-only account access, even though both live in one binary.

## Goals

- Add order preview, placement, cancelation, and amendment to `tossctl`.
- Keep the main implementation in Go.
- Reuse the existing browser-assisted login flow and session model.
- Make dangerous commands hard to run accidentally.
- Add protocol documentation and fixture coverage before implementing mutations.

## Non-Goals

- Hiding or minimizing the risk of live trading.
- Making trading commands convenient for inexperienced users.
- Supporting every order type in the first trading release.
- Supporting mobile-app-only trading flows.

## Constraints and Assumptions

- Trading will continue to target Toss Securities web flows first.
- Mutation endpoints may require more session artifacts than read-only endpoints.
- Some trading calls may depend on transient request values, device metadata, or server-issued tickets.
- Timeout or network failure after submit must be treated as ambiguous, not automatically retried.
- The project remains unofficial and subject to breakage without notice.

## Recommended Architecture

Trading support should be implemented as a separate internal subsystem layered beside the existing read-only client.

### 1. Read-Only Client Stays Separate

`internal/client` remains the read-only HTTP binding layer.

Responsibilities:

- account summary
- positions
- pending orders
- watchlist
- quotes
- session validation

Hard rule:

- no mutation endpoints in this package

### 2. Trading Client

`internal/trading` owns all mutation endpoints.

Responsibilities:

- place
- cancel
- amend
- mutation-specific error handling
- ambiguous submit handling
- post-submit reconciliation

Hard rule:

- do not mix trading request builders into the read-only client

### 3. Order Intent Layer

`internal/orderintent` converts user input into a canonical order representation.

Responsibilities:

- parse market and symbol
- normalize side and order type
- normalize quantity and price
- resolve account selection
- build stable preview payloads
- produce canonical confirmation strings

The canonical intent is the source of truth for both preview and execution.

### 4. Preflight Layer

`internal/preflight` validates whether an order can be attempted.

Responsibilities:

- local validation
- market-hours checks
- tick-size validation
- cash or position sufficiency checks
- required field checks by order type
- request-body shaping before mutation

Preflight must fail before mutation whenever there is enough information to do so safely.

### 5. Permission Gate

`internal/permissions` is the safety boundary.

Default policy:

- deny mutation

Execution requires all of the following:

- `--execute`
- `--dangerously-skip-permissions`
- `--confirm <token>`

Optional additional guard:

- short-lived permission grant with TTL

The confirmation token must be derived from the canonical preview summary. If the order intent changes, the token becomes invalid.

### 6. Audit Layer

`internal/audit` records the minimum mutation trail needed to understand what happened.

Allowed to store:

- command type
- execution timestamp
- market
- symbol
- side
- quantity
- normalized limit price
- sanitized order identifier if returned
- final reconciliation status

Not allowed to store:

- cookies
- tokens
- raw account numbers
- raw request bodies with secrets

## Command Surface

The trading command surface should be grouped under `order`.

```text
tossctl order preview --symbol TSLL --side buy --qty 10 --price 12.34 --market us --type limit
tossctl order place --symbol TSLL --side buy --qty 10 --price 12.34 --market us --type limit --execute --dangerously-skip-permissions --confirm <token>

tossctl order cancel --order-id <id> --execute --dangerously-skip-permissions --confirm <token>
tossctl order amend --order-id <id> --qty 5 --price 12.10 --execute --dangerously-skip-permissions --confirm <token>

tossctl order permissions grant --ttl 300
tossctl order permissions status
tossctl order permissions revoke
```

Design rules:

- `preview` is required before `place`
- `place`, `cancel`, and `amend` are mutation commands
- mutation commands must show a final summary before execution
- confirmation must be bound to canonical order input

## Data Flow

Trading commands should execute in the following order:

1. parse CLI arguments
2. normalize order intent
3. run preflight checks
4. render preview summary
5. evaluate permission gate
6. call trading mutation endpoint
7. reconcile by fetching follow-up order state
8. write sanitized audit record

This flow exists to prevent accidental divergence between what the user previewed and what the server executed.

## Reverse Engineering Strategy

Trading reverse engineering should be done by scenario, not by page alone.

Initial capture scenarios:

1. preview only
2. successful place
3. rejected place
4. cancel
5. amend

Recommended first market and order type:

- KR stock
- limit buy

Each scenario should capture:

- endpoint path
- method
- required headers
- request body
- transient fields
- state transitions
- success and failure response shapes
- follow-up reconciliation request pattern

The project should maintain both:

- `rpc-catalog.md`
- `order-state-machine.md`

The state-machine document is required because trading risk lives in transitions, not just endpoints.

## Failure Model

Trading errors must be categorized explicitly.

### 1. Session Failure

Examples:

- no session
- expired session
- rejected session
- extra verification required

User outcome:

- fail with re-login guidance

### 2. Input Failure

Examples:

- invalid symbol
- missing price for limit order
- invalid quantity
- unsupported market or order type

User outcome:

- fail locally before network mutation

### 3. Preflight Failure

Examples:

- insufficient buying power
- insufficient position for sell
- invalid tick size
- market closed

User outcome:

- fail before mutation

### 4. Permission Failure

Examples:

- no `--execute`
- no dangerous bypass
- invalid confirmation token
- expired permission grant

User outcome:

- fail after preview with explicit message

### 5. Broker Rejection

Examples:

- price out of range
- account restrictions
- server-side validation error

User outcome:

- show normalized rejection code and message

### 6. Ambiguous Submit

Examples:

- timeout after submit
- transport interruption
- response body missing after request commit

User outcome:

- mark status as unknown
- do not auto-retry
- instruct reconciliation via order query

## Security and Safety Model

The most important design rule is that trading should be intentionally inconvenient.

Required protections:

- mutation packages separated from read-only packages
- confirmation token bound to previewed order
- no auto-submit path
- no auto-retry after ambiguous submit
- no long-lived trading permission by default
- sanitized logging and fixtures

The README and trading docs should clearly state:

- trading mode is dangerous
- this is an unofficial client
- endpoints may change
- users are responsible for confirming what they execute

## Testing Strategy

Trading requires more than parser tests.

### Fixture Tests

Use sanitized trading fixtures to test:

- order-intent normalization
- request shaping
- rejection parsing
- reconciliation parsing
- confirmation-token matching

### Dry-Run Golden Tests

Use canonical preview inputs and expected summaries to verify:

- preview stability
- token generation stability
- permission checks

### Local Smoke Tests

Run manually against a real account with small notional values for:

- preview
- submit success
- submit rejection
- cancel
- amend

Public CI must not run live trading.

## Release Shape

The first trading release should not try to cover all markets and order types.

Recommended first capability slice:

1. KR stock limit buy preview
2. KR stock limit buy place
3. KR stock limit cancel
4. KR stock limit amend

After that:

- KR sell
- US limit
- more order types only after evidence

## Repository Additions

```text
cmd/tossctl/order.go
internal/trading/
internal/orderintent/
internal/preflight/
internal/permissions/
internal/audit/
docs/trading/
fixtures/responses/trading-sanitized/
```

## Exit Criteria

The first trading release is complete when a developer can:

1. run `tossctl order preview`
2. generate a confirmation token from the previewed order
3. place a small KR limit order only with the explicit danger gate
4. cancel and amend a live pending order
5. distinguish rejected, successful, and ambiguous outcomes
6. understand the danger model from the docs alone
