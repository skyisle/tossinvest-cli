# Toss Investment CLI Trading Expansion Implementation Plan

## Objective

Extend `tossctl` from a read-only developer CLI into a trading-capable CLI that supports preview, place, cancel, and amend while preserving a default-deny mutation model.

## Success Criteria

- A developer can preview a canonical order intent without submitting it.
- Real trading actions require `--execute`, `--dangerously-skip-permissions`, and `--confirm`.
- The CLI can place, cancel, and amend at least one supported order type end to end.
- Ambiguous submits are surfaced explicitly and are never auto-retried.
- Trading fixtures, docs, and local smoke tests exist before broader rollout.

## Stack Decision

Primary stack:

- Go for trading logic, CLI flow, permission gate, and request shaping
- existing Playwright-based login helper for session acquisition

Fallback rule:

- if trading flows depend on browser-generated transient values that cannot be replayed reliably from Go alone, a narrow helper may extract those values
- keep mutation logic out of the browser helper unless there is strong contrary evidence

## Milestones

### Milestone 6: Trading Discovery

Goal:

- document the trading protocol and state transitions before implementing mutation code

Tasks:

- capture KR limit order scenarios for preview, success, reject, cancel, and amend
- isolate mutation endpoints from read-only endpoints
- identify request headers, body shape, and transient fields
- document follow-up reconciliation requests
- create sanitized trading fixtures
- write `docs/trading/rpc-catalog.md`
- write `docs/trading/order-state-machine.md`
- write `docs/trading/error-codes.md`

Deliverables:

- trading RPC catalog
- order state machine
- sanitized trading fixtures

Exit criteria:

- each supported mutation command maps to a documented endpoint and response family
- preview and submit flows are separated clearly

### Milestone 7: Order Intent and Preview

Goal:

- create a stable canonical model for trading input and dry-run output

Tasks:

- add `internal/orderintent`
- add `internal/preflight`
- add `cmd/tossctl/order.go`
- implement `tossctl order preview`
- normalize market, side, quantity, price, and account selection
- render canonical preview output
- generate deterministic confirmation-token input

Deliverables:

- canonical order-intent model
- preview command
- local input validation

Exit criteria:

- preview runs without mutation
- preview output matches canonical request intent
- invalid input fails before network mutation

### Milestone 8: Permission Gate

Goal:

- prevent accidental mutation by design

Tasks:

- add `internal/permissions`
- require `--execute`
- require `--dangerously-skip-permissions`
- require `--confirm`
- optionally add short-lived permission grants with TTL
- reject mismatched confirmation tokens
- bind confirmation tokens to canonical preview summaries

Deliverables:

- permission gate package
- mutation guard integration in CLI
- token verification tests

Exit criteria:

- mutation commands cannot run without explicit approval inputs
- changing order inputs invalidates previous confirmation tokens

### Milestone 9: Place

Goal:

- implement the first safe end-to-end order placement flow

Initial scope:

- KR stock limit buy

Tasks:

- add `internal/trading/place.go`
- build submit request from canonical order intent
- classify broker rejection responses
- classify ambiguous submit conditions
- reconcile placed order against follow-up order queries
- add local smoke script for small test orders

Deliverables:

- place command
- normalized order-submit result model
- reconciliation logic

Exit criteria:

- one small live KR limit order can be placed successfully
- at least one rejected order path is reproduced and classified
- ambiguous submit path does not auto-retry

### Milestone 10: Cancel and Amend

Goal:

- complete the pending-order lifecycle for the first supported order type

Tasks:

- add `internal/trading/cancel.go`
- add `internal/trading/amend.go`
- normalize server order identifiers
- verify cancel eligibility
- verify amendable fields
- reconcile post-cancel and post-amend states

Deliverables:

- cancel command
- amend command
- follow-up reconciliation logic

Exit criteria:

- a live pending order can be canceled
- a live pending order can be amended
- already-completed or invalid orders fail with normalized errors

### Milestone 11: Hardening and OSS Readiness

Goal:

- make the trading flow safe enough to publish to developer users

Tasks:

- add `internal/audit`
- redact trading logs and fixtures
- add golden tests for preview output
- add tests for confirmation tokens and permission failures
- add smoke scripts for place, reject, cancel, and amend
- document the danger model in README
- document break-fix workflow when trading endpoints change

Deliverables:

- contributor-facing docs
- mutation safety tests
- sanitized audit behavior

Exit criteria:

- trading mode docs are sufficient for another developer to reproduce the flow
- dangerous commands are guarded and tested

## Work Breakdown by Area

### Protocol and Discovery

- scenario-based capture
- request/response cataloging
- transient-field identification
- rejection-code mapping

### CLI and UX

- preview rendering
- mutation confirmation prompts
- danger-flag semantics
- consistent error messages

### Trading Core

- canonical order intent
- submit/cancel/amend request builders
- reconciliation flows
- ambiguous-status handling

### Safety

- permission gating
- confirmation tokens
- audit trail
- redaction

## Risks and Mitigations

### Risk: Trading requires extra transient values not visible in read-only calls

Mitigation:

- capture full scenario flows
- document browser-derived values explicitly
- isolate any helper-only extraction to narrow interfaces

### Risk: Endpoint changes can cause accidental malformed orders

Mitigation:

- keep RPC catalog current
- require preview to match execution intent
- keep tests tied to sanitized fixtures

### Risk: Timeout causes duplicate orders

Mitigation:

- never auto-retry mutation
- always reconcile after uncertain outcomes
- classify ambiguous submits separately

### Risk: Safety flags become easy to bypass

Mitigation:

- require multiple independent approval inputs
- bind confirmation to canonical order content
- keep permission grants short-lived

## Suggested Task Order

1. Capture KR limit order scenarios.
2. Write trading RPC and state-machine docs.
3. Build canonical order-intent and preview flow.
4. Add permission gate and confirmation tokens.
5. Implement first place flow.
6. Implement cancel and amend.
7. Add hardening, docs, and smoke tests.

## Definition of Done

Trading support is ready for the first controlled release when:

- `tossctl order preview` is stable and canonical
- `place`, `cancel`, and `amend` are gated behind explicit danger approval
- KR limit-order flows are documented and tested
- ambiguous submits are handled without auto-retry
- README and trading docs explain the risk model clearly
