# tossinvest-cli Beta Closure Design

Date: 2026-03-13
Status: Approved
Scope: post-mutation lineage and verification closure for the current trading beta

## Goal

현재 지원 중인 `US buy limit / KRW / non-fractional` 거래 베타를, mutation 이후 상태 추적까지 포함해 신뢰 가능한 수준으로 닫는다.

이번 단계는 새 주문 타입을 추가하지 않는다. 대신 `place`, `cancel`, `amend` 이후 사용자가 어떤 주문 ref를 다시 봐야 하는지 CLI가 일관되게 안내하고, live verification 결과를 문서와 출력에 맞춘다.

## In Scope

- `order amend` live 재검증
- `cancel`과 `amend` 이후 order ref rollover 추적
- `order show`의 post-mutation lookup 개선
- verification note, README, trading docs 정합성 갱신

## Out of Scope

- `sell`
- `market`
- `KR`
- `fractional`
- `dangerous_automation` runtime handler 구현
- 새 거래 명령 추가

## Approach Options

### 1. Reconciliation-first Closure

먼저 mutation 이후 surviving order를 어떻게 추적할지 닫고, 그 다음 live evidence와 문서를 정리한다.

장점:

- 현재 베타에서 가장 위험한 "실행은 됐는데 어떤 주문을 봐야 하는지 모른다" 문제를 직접 해결한다
- `order show`와 mutation 결과가 같은 lineage 모델을 공유하게 된다
- interactive auth 분기가 남아 있어도 CLI의 신뢰도가 오른다

단점:

- 위험한 자동화 자체는 아직 도입하지 않는다

### 2. Automation-first Closure

`dangerous_automation.complete_trade_auth`를 먼저 구현해서 amend end-to-end 성공률을 높인다.

장점:

- `amend` 성공 표면이 빨리 넓어진다

단점:

- 아직 lineage가 덜 닫힌 상태에서 위험한 자동화가 먼저 들어간다
- verification보다 complexity가 앞선다

### 3. Docs-first Closure

코드는 최소만 손보고 README와 verification note를 우선 정리한다.

장점:

- 가장 빨리 끝난다

단점:

- 사용자 신뢰도는 거의 오르지 않는다

## Recommendation

`Reconciliation-first Closure`로 간다.

지금 필요한 것은 breadth보다 traceability다. `amend`와 `cancel`은 mutation 이후 order identity가 바뀔 수 있기 때문에, 먼저 CLI가 surviving order를 다시 찾을 수 있어야 한다.

## Execution Shape

### 1. Lineage and Reconciliation

- mutation 결과에서 `original -> current` order id rollover를 보존한다
- local lineage cache를 통해 `order show <old-ref>`가 새 ref를 다시 찾을 수 있게 한다
- `cancel`은 pending disappearance만 보는 대신 completed history도 같이 읽어 rollover를 확인한다

### 2. Live Closure and Docs

- `amend`를 실제로 다시 검증한다
- 결과가 success인지 interactive auth인지 문서로 남긴다
- README와 doctor를 현재 evidence 기준으로 맞춘다

## Deliverables

- lineage cache service
- `cancel` reconciliation 개선
- `order show` alias fallback
- regression tests
- updated verification note and README

## Success Criteria

- `cancel` 또는 `amend` 이후 order ref가 바뀌어도 사용자가 CLI만으로 최종 주문을 다시 찾을 수 있다
- `order show <original-ref>`가 same-machine lineage cache를 통해 surviving order를 찾을 수 있다
- `amend`는 success 또는 interactive auth required 중 실제 outcome이 문서에 남는다
- README와 doctor가 현재 지원 범위와 미검증 범위를 과장하지 않는다
