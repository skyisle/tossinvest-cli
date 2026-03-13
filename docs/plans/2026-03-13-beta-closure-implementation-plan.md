# tossinvest-cli Beta Closure Implementation Plan

Date: 2026-03-13
Status: Drafted from approved design
Scope: lineage cache, cancel reconciliation, order lookup fallback, and docs

## Objective

현재 trading beta의 mutation 이후 order rollover를 CLI가 다시 추적할 수 있게 만들고, 그 동작을 live verification 결과와 문서에 맞춘다.

## Phase 1. Local Lineage Tracking

### Work

- local lineage cache file path를 추가한다
- `original_order_id -> current_order_id` mapping을 저장하는 service를 만든다
- `order show`가 exact lookup 실패 시 lineage cache를 사용해 successor id를 다시 찾게 한다

### Deliverables

- new lineage service under `internal/`
- root/app wiring for lineage cache
- order lookup regression tests

## Phase 2. Cancel and Amend Reconciliation

### Work

- `cancel` reconciliation을 pending disappearance + completed-history lookup까지 확장한다
- cancel result에 rollover된 current order id를 채운다
- amend/cancel 결과를 lineage cache에 기록한다

### Deliverables

- updated `internal/client/trading.go`
- updated `internal/trading/service.go`
- client and service tests

## Phase 3. Output and Docs

### Work

- `order show` output에 alias resolution 정보를 노출한다
- README support matrix와 known limitation을 갱신한다
- verification note와 trading docs를 live evidence 기준으로 갱신한다
- doctor scope wording을 현재 검증 상태 기준으로 맞춘다

### Deliverables

- updated `internal/output`
- updated `internal/doctor`
- updated docs

## Verification

- `go test ./...`
- `make build`
- `tossctl doctor`
- live retest for `order amend`
- live retest for `order show <original-ref>` after cancel/amend rollover

## Risks

- lineage cache는 같은 로컬 config dir에서 실행한 mutation에 대해서만 alias fallback을 제공한다
- completed history 반영이 지연되면 cancel reconciliation은 warning과 함께 current id 없이 끝날 수 있다
- amend live retest는 broker-side interactive auth에 다시 막힐 수 있다

## Expected Outcome

사용자는 현재 trading beta 범위에서 다음을 할 수 있다.

- mutation 후 surviving order ref를 다시 찾는다
- `order show <old-ref>`로 새 ref를 따라간다
- cancel/amend rollover가 CLI 출력과 문서에서 일관되게 설명된다
