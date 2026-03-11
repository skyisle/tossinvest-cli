# Trading Error Codes

This file will map Toss Securities trading rejection and failure signals to normalized CLI errors.

Current status:

- successful fill, pending acceptance, bulk cancel, and amend success paths captured
- first server-side rejection path observed at `order/prepare`

## Current Known Failure Classes

### Session Failures

Already normalized in the CLI:

- no active session
- stored session rejected

These remain valid for trading mode as well.

### Preview and Entry Preconditions

Observed endpoints that likely gate preview and submit:

- `GET /api/v2/trading/order/{stockCode}/prerequisite`
- `GET /api/v3/trading/order/{stockCode}/trading-status`
- `GET /api/v1/trading/orders/calculate/{stockCode}/orderable-quantity/sell`
- `GET /api/v2/trading/orders/calculate/{stockCode}/cost-basis-elements`

Expected future normalized failures from these classes:

- market not tradable
- account not eligible
- insufficient sellable quantity
- unsupported order type for current market
- product-risk acknowledgement required

### Buying-Power and Quantity Signals

Observed UI signals:

- buy-side page displayed `구매가능 금액 14원`
- sell-side quantity field displayed `최대 114주 가능`

Expected future normalized failures:

- insufficient buying power
- quantity exceeds sellable shares
- fractional mode mismatch

### Product Risk Gates

Observed on TSLL:

- clicking `구매하기` opened a leveraged or inverse ETP risk notice dialog
- no submit mutation was observed before the dialog

Expected normalized class:

- `product_ack_required`

CLI implication:

- preview and preflight must be able to surface product-risk acknowledgement requirements separately from broker rejection

### Prepare-Stage Buying Power Failure

Observed on TSLL buy flow:

- quantity `1`
- `구매하기`
- leveraged ETP acknowledgement
- `POST /api/v2/wts/trading/order/prepare` -> `422 Unprocessable Entity`

Observed paired checks:

- `GET /api/v1/account/investment-propensity/eligible/with-contract?financialQualification=USA_STOCK&highRiskTradingCategory=ETF`
- `GET /api/v1/product-eligibility/overseas-leverage-etp/trading`

Observed UI rejection:

- title: `계좌 잔액이 부족해요`
- body: `구매를 위해 21,511원을 채울게요.`
- actions:
  - `닫기`
  - `모바일에서 채우기`

Expected normalized class:

- `insufficient_buying_power`

CLI implication:

- `order preview` can remain local or calculator-backed
- `order place` must treat `prepare` failures as broker-side rejections and surface the user-facing amount gap when available
- successful submit logic must start only after `prepare` succeeds
- some retries may depend on user-driven funding and FX approval outside the immediate order form

### Successful Immediate Fill

Observed on TSLL:

- `POST /api/v2/wts/trading/order/prepare` -> `200`
- `POST /api/v2/wts/trading/order/create` -> `200`

Observed UI:

- `TSLL 구매 완료`
- `1주 가격 21,208원`
- `구매 수량 1주`
- `총 구매 금액 21,208원`

Observed reconciliation:

- no pending order remained
- completed history exposed the row `3.11 TSLL 주당 21,208원 1주 구매 완료`

User-reported branch between failure and success:

- mobile balance top-up notification
- browser-side FX approval prompt
- then successful retry

CLI implication:

- success cannot be inferred from `pending` alone
- reconciliation must check completed history in addition to pending state
- a production trading client will likely need to classify `funding_required` and `fx_consent_required` separately from plain `insufficient_buying_power`
- the normalized success model needs at least:
  - `accepted_pending`
  - `filled_immediately`
  - `accepted_but_unknown`

### Successful Pending Acceptance

Observed on TSLL:

- low-price limit order was accepted
- CLI reflected:
  - `status: 체결대기`
  - `pendingQuantity: 1`
  - `orderPrice: 1000`
- account summary reflected `pending_buy_order_amount: 1001`

CLI implication:

- a successful `create` can resolve to `accepted_pending` rather than `filled_immediately`
- order reconciliation should normalize and preserve:
  - pending quantity
  - limit price
  - correction support flags
- captured place payload for this path also showed:
  - `market: NSQ`
  - `currencyMode: KRW`
  - `price` sent in USD precision
  - `allowAutoExchange: true`
  - `marginTrading: false`
  - `extra.*` present only on the final `create` step

### Bulk Cancel Success

Observed on TSLL pending order:

- `POST /api/v3/wts/trading/order/bulk-cancel/prepare` -> `200`
- `POST /api/v3/wts/trading/order/bulk-cancel` -> `200`
- UI changed to `대기중인 주문이 없어요`
- CLI `orders list` changed to `[]`

CLI implication:

- cancel flow should mirror place flow with a prepare stage
- the initial cancel implementation can target bulk-cancel first
- reconciliation after cancel should read pending history and confirm the order is gone

### Single-Order Cancel Success

Observed on the amended TSLL pending order:

- `GET /api/v3/trading/order/5/available-actions?...` -> `200`
- `POST /api/v2/wts/trading/order/cancel/prepare/2026-03-11/5` -> `200`
- `POST /api/v3/wts/trading/order/cancel/2026-03-11/5` -> `200`
- UI changed to `대기중인 주문이 없어요`
- CLI `orders list` changed to `[]`

CLI implication:

- inline cancel is distinct from bulk cancel and should be modeled separately
- both cancel modes still follow `prepare -> mutation -> reconcile`
- mutation success alone is insufficient; the client should confirm the order disappeared from pending history
- a later capture confirmed the cancel body includes:
  - `stockCode`
  - `tradeType`
  - `quantity`
  - `isAfterMarketOrder`
  - `isReservationOrder`
  - `withOrderKey` only on the prepare step

### Amend Success

Observed on TSLL pending order:

- `GET /api/v3/trading/order/4/available-actions?...` -> `200`
- `POST /api/v2/wts/trading/order/correct/prepare/2026-03-11/4` -> `200`
- `POST /api/v2/wts/trading/order/correct/2026-03-11/4` -> `200`
- pending-history reconciliation showed the order still open at the amended price

Observed post-amend state:

- `orderPrice: 500`
- `pendingQuantity: 1`
- `status: 체결대기`
- `pending_buy_order_amount: 500`

CLI implication:

- amend should be modeled as `prepare -> correct -> reconcile`
- amend success does not mean finality; the order remains mutable and pending
- the amend path key did not match the later CLI-visible pending order number, so the client should assume order identity may roll forward after correction
- amend completion must always re-read pending history and remap the surviving order record
- captured amend payload also showed:
  - `market: NSQ`
  - `currencyMode: KRW`
  - `price` expressed in USD precision
  - `withOrderKey` only on the prepare step

### Ambiguous Submit

Not yet captured, but must map to a separate error family:

- request committed but response missing
- timeout after submit
- connection drop during mutation

CLI rule:

- never auto-retry on ambiguous submit
- always reconcile through order-history fetch

## To Capture Next

- rejection body for `order/prepare` if accessible via browser-side fetch interception
- successful `order/prepare` response
- successful `order/create` response
- funding and FX approval branch requests
- any structured broker error codes
- submit-time validation vs preflight validation differences
- amend rejection cases
