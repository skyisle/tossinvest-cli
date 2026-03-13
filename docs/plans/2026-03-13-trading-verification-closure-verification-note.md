# tossinvest-cli Trading Verification Closure Verification Note

Date: 2026-03-13
Status: Live verification executed
Scope: US buy limit / KRW / non-fractional only

## Completed Checks

### Automated

- `go test ./...`
  - result: pass
- `make build`
  - result: pass

### Safe CLI Readiness

- `tossctl version`
  - result: pass
- `tossctl doctor`
  - result: pass
  - session file exists
  - trading permission file exists but temporary permission is expired
  - config file does not exist yet, so trading actions default to disabled
- `tossctl auth doctor`
  - result: pass
  - auth helper importable
  - playwright installed
  - chromium installed
  - stored session valid
- `tossctl auth status`
  - result: active session
  - provider: `playwright-storage-state`
  - live check: valid

### Read-only Order Visibility

- `tossctl orders list --output json`
  - result: pass
  - observed state: no pending orders at the time of check
- `tossctl orders completed --market us --output json`
  - result: pass
  - observed state: completed-history lookup works for current month US orders
  - observed statuses in history: `체결완료`, `취소`, `실패`
- `tossctl order show 2026-03-11/25 --market us --output json`
  - result: pass
  - observed state: canceled order lookup works through the single-order surface
- `tossctl order show 2026-03-11/1 --market us --output json`
  - result: pass
  - observed state: completed order lookup works through the single-order surface
- `tossctl order preview --symbol TSLL --market us --side buy --type limit --qty 1 --price 500 --currency-mode KRW --output json`
  - result: pass
  - observed state: preview emits canonical intent and confirm token
  - observed state: `live_ready=true`, `mutation_ready=false` while config remains disabled

## Current Blockers for Live Mutation Verification

- none for basic execution readiness

## Live Execution Results

### Live place

- command: `tossctl order place --symbol TSLL --market us --side buy --type limit --qty 1 --price 500 --currency-mode KRW --execute ...`
- result: success
- returned mutation status: `accepted_pending`
- returned order reference: `2026-03-13/1`
- follow-up:
  - `orders list` showed the order as `체결대기`
  - `order show 2026-03-13/1` also resolved the pending order

### Live amend

- command target: pending order `2026-03-13/1`, new price `700 KRW`
- first observed issue:
  - the implementation sent the user-facing order reference into `available-actions`
  - broker returned `404`
- second observed issue:
  - even with the broker raw order id, the value was not path-escaped, which also caused `404`
- code fix applied during verification:
  - resolve the raw broker order id from the pending order payload
  - path-escape that id for `available-actions`
  - treat `400` and `404` from `available-actions` as soft preflight failure so the real mutation path can continue
- post-fix live result:
  - mutation reached the broker but stopped with `interactive trade authentication required`
- conclusion:
  - `amend` is not yet end-to-end verified for this account/session
  - the path construction bug is fixed
  - the remaining blocker is broker-side interactive auth

### Live cancel

- command target: pending order `2026-03-13/1`
- result: success
- returned mutation status: `canceled`
- follow-up:
  - `orders list` became empty
  - completed history did not keep the original reference `2026-03-13/1`
  - completed history recorded the canceled order as `2026-03-13/2`
  - `order show 2026-03-13/2` resolved the canceled order successfully
  - later code change added a local lineage cache so `order show <original-id>` can follow this rollover when the mutation was executed through the same config dir

## Code Changes Found Necessary During Live Verification

- `GetOrderAvailableActions` now uses the resolved raw pending-order id instead of the user-facing reference id
- the broker raw id is path-escaped before calling `available-actions`
- `400` and `404` responses from `available-actions` are treated as soft failures because cancel/amend do not consume that payload today
- interactive-auth user-facing text now refers to the generic trade action instead of saying "cancel" unconditionally
- regression tests added for:
  - resolved broker order id path construction
  - path escaping of raw order ids
  - soft-failure handling for `400`
  - cancel completed-history rollover reconciliation
  - local alias lookup for `order show`

## Post-Run Safety State

- trading permission revoked
- local `config.json` restored to all trading flags disabled

## Still Pending

- live re-test of `order amend` after lineage/reconciliation changes
- evidence-driven confirmation of whether the observed interactive-auth branch for `amend` is account-specific or generally expected

## Next Operator Steps

1. Run full `go test ./...` after the live-driven fixes.
2. Live re-test `order amend` and record whether the outcome is still `interactive auth required` or a completed success path.
3. If needed, document any new amend-specific rollover evidence in trading docs.
