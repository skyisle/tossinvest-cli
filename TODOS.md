# TODOS

## Sell 주문 live 검증 확대

- **What:** 분할 매도(일부 수량), 전량 매도, 보유량 초과 요청 시 API 동작 검증
- **Why:** 분할 매도는 sell의 일반 케이스. prerequisite dry-run에서 기본 happy-path는 확인하지만, 수량 관련 edge case는 별도 live 검증 필요.
- **Pros:** sell 기능의 신뢰도 확보, 사용자가 예상치 못한 에러 방지
- **Cons:** 실제 보유 주식이 있어야 검증 가능 (테스트 비용)
- **Context:** Codex 리뷰에서 "partial sell is the normal case" 지적 (2026-03-21). FX consent 방향(USD→KRW), auth-required 분기, holdings rejection 등도 함께 검증 권장.
- **Depends on:** sell order 구현 + 첫 live sell 검증 완료 후

## KR stock cancel/amend live 검증

- **What:** 국내주식 cancel/amend에서 `InferMarketFromStockCode`가 올바르게 동작하는지 live 확인
- **Why:** cancel/amend reconciliation이 market을 stockCode 패턴에서 추론하므로, 실제 KR 주문 cancel/amend가 올바른 시장에서 reconcile되는지 검증 필요.
- **Pros:** KR cancel/amend 신뢰도 확보
- **Cons:** 실제 KR 미체결 주문이 있어야 검증 가능
- **Context:** KR trading eng review에서 Codex가 지적 (2026-03-21). `pendingOrderDetails`에 market 필드가 없어 stockCode 패턴 추론 사용.
- **Depends on:** KR trading 구현 + 첫 live KR 주문 완료 후

## Completed

### Reconciliation market 파라미터화
- **Completed:** v0.1.7 (2026-03-21)
- reconcilePlacedOrder, reconcileAmendedOrder, reconcileCanceledOrder의 `Market: "us"` 하드코딩 8곳을 market 파라미터로 전환. cancel/amend는 `InferMarketFromStockCode`로 market 추론.
