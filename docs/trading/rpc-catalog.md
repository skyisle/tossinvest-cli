# Trading RPC Catalog

Verified from the TSLL order page on 2026-03-11.

Current capture scope:

- `preview-only`
- authenticated web session
- stock page: `/stocks/US20220809012/order`
- market: `us`

## Status Legend

- `observed`: seen in authenticated browser traffic
- `captured`: request and response family confirmed during a directed scenario
- `unknown`: likely relevant but not yet isolated

## Preview and Order-Entry Endpoints

| Status | Method | Host | Path | Purpose | Notes |
| --- | --- | --- | --- | --- | --- |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v3/trading/order/US20220809012/trading-status` | product trading status | loaded on page entry before order interaction |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v2/trading/order/US20220809012/prerequisite` | order-entry prerequisites | re-fetched when switching to `판매` mode |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/orders/calculate/US20220809012/orderable-quantity/sell?forceFetch=false` | sellable quantity lookup | loaded on entry and again during sell-mode rendering |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v2/trading/orders/calculate/US20220809012/cost-basis-elements` | average cost and sell-side basis data | paired with sell-mode preview widgets |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/orders/calculate/US20220809012/average-price?forceFetch=false` | average price lookup | used on order page initialization |
| `captured` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/orders/calculate/order-data` | buy-side preview calculation | two POSTs observed on page load with different `orderPrice` values |
| `captured` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v2/wts/trading/order/prepare` | first live pre-submit mutation candidate | observed only after leveraged ETP acknowledgement; returned `422` on insufficient funds |
| `captured` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v2/wts/trading/order/create` | live order creation mutation | observed after a successful `order/prepare`; returned `200` in the filled TSLL buy flow |
| `captured` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v3/wts/trading/order/bulk-cancel/prepare` | bulk-cancel preflight mutation | observed when the user clicked `전체 취소` on the pending-order panel |
| `captured` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v3/wts/trading/order/bulk-cancel` | bulk-cancel live mutation | returned `200` and cleared the pending TSLL order |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v3/trading/order/4/available-actions?stockCode=US20220809012&tradeType=buy&orderPriceType=00&fractional=false&isReservationOrder=false` | pending-order amend capability lookup | observed immediately before the user-driven `수정` flow |
| `captured` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v2/wts/trading/order/correct/prepare/2026-03-11/4` | amend preflight mutation | returned `200` before the TSLL pending order was repriced from `1,000원` to `500원` |
| `captured` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v2/wts/trading/order/correct/2026-03-11/4` | amend live mutation | returned `200` and left the order pending at the new price |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v3/trading/order/5/available-actions?stockCode=US20220809012&tradeType=buy&orderPriceType=00&fractional=false&isReservationOrder=false` | pending-order cancel capability lookup | observed immediately before the inline `취소` flow after amend |
| `captured` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v2/wts/trading/order/cancel/prepare/2026-03-11/5` | single-order cancel preflight mutation | returned `200` when the user clicked inline `취소` on the amended TSLL order |
| `captured` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v3/wts/trading/order/cancel/2026-03-11/5` | single-order cancel live mutation | returned `200` and cleared the pending TSLL order |
| `observed` | `GET` | `wts-api.tossinvest.com` | `/api/v1/trading/settings/toggle/find?categoryName=TRADE_WITHOUT_CONFIRM` | fetch trade-without-confirm preference | may become relevant for permission UX mapping |
| `captured` | `GET` | `wts-api.tossinvest.com` | `/api/v1/trading/settings/toggle/find?categoryName=GETTING_BACK_KRW` | fetch FX-return preference before the browser FX prompt | observed only after the confirmation-dialog `구매` click on 2026-03-13 |
| `captured` | `GET` | `wts-api.tossinvest.com` | `/api/v1/exchange/current-quote/for-buy` | fetch live exchange quote for the FX-confirmation modal | returned `rateQuoteId`, `usdRate`, and validity window immediately before the FX prompt on 2026-03-13 |
| `observed` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v2/trading/settings/investor-exchange-choice-type` | exchange-choice setting | likely affects market routing |
| `captured` | `GET` | `wts-api.tossinvest.com` | `/api/v1/account/investment-propensity/eligible/with-contract?financialQualification=USA_STOCK&highRiskTradingCategory=ETF` | overseas leverage eligibility check | fetched immediately before `order/prepare` |
| `captured` | `GET` | `wts-api.tossinvest.com` | `/api/v1/product-eligibility/overseas-leverage-etp/trading` | overseas leverage product eligibility check | fetched immediately before `order/prepare` |
| `captured` | `POST` | `wts-api.tossinvest.com` | `/api/v1/trading/settings/toggle` | exchange-confirmation acknowledgement update | observed on FX-modal `확인` click with `{"categoryName":"EXCHANGE_INFO_CHECK","turnedOn":true}` |

## Order History and Sidecar Endpoints

| Status | Method | Host | Path | Purpose | Notes |
| --- | --- | --- | --- | --- | --- |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/orders/histories/all/pending` | global pending-order summary | loaded with the trading page shell |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/orders/histories/PENDING?stockCode=US20220809012&number=1&size=100&marketDivision=us` | symbol-specific pending orders | shown in the order page side panel |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/orders/histories/COMPLETED?stockCode=US20220809012&number=1&size=30&marketDivision=us` | symbol-specific completed orders | loaded on entry |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v2/trading/my-orders/markets/us/by-date/completed?range.from=2026-02-28&range.to=2026-03-31&size=20&number=1` | completed US order history for the month view | fetched when switching the `주문내역` panel to `완료` |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v2/trading/my-orders/markets/kr/by-date/completed?range.from=2026-03-01&range.to=2026-03-31&size=20&number=1` | completed KR order history for the month view | fetched alongside the US completed-history query |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v4/trading/auto-trading?productCode=US20220809012&size=20&number=1` | auto-trading metadata | likely informational, not needed for first mutation path |
| `captured` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/analysis/productCode/US20220809012` | trading analysis panel data | informational only so far |

## Captured `order-data` Request Bodies

Two buy-side preview POST bodies were captured during page load:

```json
{
  "stockCode": "US20220809012",
  "market": "us",
  "orderPrice": 0,
  "orderVolumeRate": 1,
  "currencyMode": "KRW",
  "isFractional": false
}
```

```json
{
  "stockCode": "US20220809012",
  "market": "us",
  "orderPrice": 21134,
  "orderVolumeRate": 1,
  "currencyMode": "KRW",
  "isFractional": false
}
```

Current inference:

- `orderPrice=0` is likely a market-price or placeholder calculation path.
- `orderPrice=21134` matches the default limit price shown in the buy panel.
- `orderVolumeRate=1` appears to represent full entered quantity in the current UI model.
- `currencyMode=KRW` is explicitly sent even on a US stock order page.

`order/prepare` request body has not been isolated yet, but its position in the sequence is now clear:

- quantity entry
- `구매하기`
- leveraged or inverse ETP acknowledgement
- eligibility checks
- `POST /api/v2/wts/trading/order/prepare`

## UI Observations Tied To Preview

### Buy-side initial state

- mode: `구매`
- price mode: `지정가`
- price field defaulted to `21,134`
- quantity empty on initial load
- buying power shown as `14원`
- total order amount updated locally when quantity was entered

### Sell-side preview state

- mode: `판매`
- price mode: `지정가`
- price field showed `21,119`
- quantity `1` displayed placeholder `최대 114주 가능`
- UI summary showed:
  - `현재 수익`
  - `예상 수익률`
  - `예상 손익`
  - `총 금액`

Sell-mode switching triggered at least:

- `GET /api/v2/trading/order/US20220809012/prerequisite`
- `GET /api/v2/trading/orders/calculate/US20220809012/cost-basis-elements`

## Pre-Submit Gating Observations

Attempted action:

- buy mode
- quantity `1`
- clicked `구매하기`

Observed result:

- no live submit mutation was captured
- a blocking dialog appeared for leveraged or inverse ETP risk disclosure

Dialog summary:

- title family: leveraged or inverse ETP risk notice
- action button: `확인했어요`

Current inference:

- certain products require a product-risk acknowledgement step before any actual place mutation
- this gate must be modeled separately from permission flags and preview calculation

### Post-Acknowledgement Insufficient-Balance Gate

Attempted continuation:

- clicked `확인했어요` on the leveraged ETP notice

Observed result:

- `GET /api/v1/account/investment-propensity/eligible/with-contract?financialQualification=USA_STOCK&highRiskTradingCategory=ETF`
- `GET /api/v1/product-eligibility/overseas-leverage-etp/trading`
- `POST /api/v2/wts/trading/order/prepare` -> `422 Unprocessable Entity`
- UI alert:
  - title: `계좌 잔액이 부족해요`
  - body: `구매를 위해 21,511원을 채울게요.`
  - actions:
    - `닫기`
    - `모바일에서 채우기`

Current inference:

- `order/prepare` is the first live server-side mutation candidate after product acknowledgement
- insufficient buying power is enforced at `prepare`, not only through local preview widgets
- there is likely at least one additional step after `prepare` for successful orders, because this path stops at a broker-side blocking alert rather than creating a pending order
- when funds are insufficient, the user may be redirected into a top-up path outside the order panel before retrying `prepare`

### Post-Prepare FX Prompt Path

Observed on 2026-03-13 in a headed desktop web replay of `TSLL` 1주 `1000 KRW`:

- first `구매하기` click:
  - `POST /api/v2/wts/trading/order/prepare` -> `200 OK`
  - response body included `preparedOrderInfo.needExchange: 0.68`
  - UI showed only the normal order confirmation dialog
- confirmation-dialog `구매` click:
  - `GET /api/v1/trading/settings/toggle/find?categoryName=GETTING_BACK_KRW` -> `200 OK`
  - `GET /api/v1/exchange/current-quote/for-buy` -> `200 OK`
  - UI then showed:
    - `0.68달러가 부족해요`
    - `주식 구매를 위해 환전할게요`
    - `주문이 취소되면 계좌에는 달러로 남아있어요.`
- no `order/create` request was observed before that FX modal
- FX-modal `확인` click:
  - `POST /api/v2/wts/trading/order/create`
  - `POST /api/v1/trading/settings/toggle`
  - the toggle body was `{"categoryName":"EXCHANGE_INFO_CHECK","turnedOn":true}`

Current inference:

- the desktop web FX prompt is not a plain `prepare` rejection
- it is a post-prepare confirmation branch that depends on `needExchange` and an exchange-quote fetch
- the actual confirmation step is not a separate FX-mutation RPC; it is the same `order/create` mutation gated behind the FX modal plus a sidecar toggle update

### Successful Filled Buy Path

User-driven action:

- a manual TSLL buy order was submitted in the browser and filled
- user-reported path:
  - insufficient-balance alert
  - mobile push for balance top-up
  - return to browser
  - browser-side foreign-exchange confirmation
  - order submission and fill

Observed result:

- `POST /api/v2/wts/trading/order/prepare` -> `200 OK`
- `POST /api/v2/wts/trading/order/create` -> `200 OK`
- `POST /api/v1/trading/settings/toggle` -> `200 OK`
- immediate reconciliation fetches:
  - `GET /api/v1/trading/orders/histories/all/pending`
  - `GET /api/v1/trading/orders/histories/PENDING?...`
  - `GET /api/v1/trading/orders/calculate/US20220809012/orderable-quantity/sell?forceFetch=true`
  - `GET /api/v1/trading/orders/calculate/US20220809012/average-price?forceFetch=true`

Observed UI:

- dialog title: `TSLL`
- dialog status: `구매 완료`
- fields:
  - `1주 가격`: `21,208원`
  - `구매 수량`: `1주`
  - `총 구매 금액`: `21,208원`
- `주문내역 > 완료` row:
  - `3.11 TSLL 주당 21,208원 1주 구매 완료`

Post-trade account observation:

- TSLL holdings increased from `114.192125주` to `115.192125주`
- no pending order remained after reconciliation

Current inference:

- the successful live order path is `prepare -> create -> reconciliation`
- in the insufficient-balance case, successful retry may depend on an external top-up step and an explicit browser-side FX approval step before `create`
- the order appears to have filled immediately, so the CLI cannot rely on pending orders alone to confirm success
- completed-order history queries are required for reliable post-submit reconciliation

### Captured Place Request Bodies

For a later low-price TSLL buy that remained pending, the browser emitted:

`order/prepare` body:

```json
{
  "stockCode": "US20220809012",
  "market": "NSQ",
  "currencyMode": "KRW",
  "tradeType": "buy",
  "price": 0.3395,
  "quantity": 1,
  "orderAmount": 0,
  "orderPriceType": "00",
  "agreedOver100Million": false,
  "marginTrading": false,
  "max": false,
  "allowAutoExchange": true,
  "isReservationOrder": false,
  "openPriceSinglePriceYn": false,
  "withOrderKey": true
}
```

`order/create` body:

```json
{
  "stockCode": "US20220809012",
  "market": "NSQ",
  "currencyMode": "KRW",
  "tradeType": "buy",
  "price": 0.3395,
  "quantity": 1,
  "orderAmount": 0,
  "orderPriceType": "00",
  "agreedOver100Million": false,
  "marginTrading": false,
  "max": false,
  "allowAutoExchange": true,
  "isReservationOrder": false,
  "openPriceSinglePriceYn": false,
  "extra": {
    "close": 14.4,
    "closeKrw": 21208,
    "exchangeRate": 1472.8,
    "orderMethod": "종목상세__주문하기"
  }
}
```

Additional observations from the same place:

- pending order after reconciliation became:
  - `orderNo: 16`
  - `orderPrice: 500`
  - `orderUsdPrice: 0.3395`
  - `status: 체결대기`
- `withOrderKey` appears on `prepare` but not on `create`
- `extra.closeKrw` matched the captured exchange rate and USD close:
  - `14.4 * 1472.8 ≈ 21208`

### Pending Order And Bulk Cancel Path

User-driven action:

- a manual TSLL buy order was submitted at a very low price and remained pending
- the user then clicked `전체 취소`

Observed pending state:

- CLI `orders list` surfaced:
  - `stockCode: US20220809012`
  - `stockName: TSLL`
  - `orderPrice: 1000`
  - `quantity: 1`
  - `pendingQuantity: 1`
  - `status: 체결대기`
- account summary reflected `pending_buy_order_amount: 1001` in the US market
- order page `주문내역 > 대기` showed:
  - `TSLL`
  - `주당 1,000원`
  - `1주 구매`
  - action buttons `수정`, `취소`
  - page-level action `전체 취소`

Observed cancel result:

- `POST /api/v3/wts/trading/order/bulk-cancel/prepare` -> `200 OK`
- `POST /api/v3/wts/trading/order/bulk-cancel` -> `200 OK`
- follow-up reconciliation:
  - repeated `GET /api/v1/trading/orders/histories/PENDING?...`
- UI after cancel:
  - `대기중인 주문이 없어요`
- CLI after cancel:
  - `orders list` returned `[]`

Current inference:

- Toss supports a dedicated bulk-cancel flow distinct from `order/create`
- even a single pending order can be canceled through the bulk-cancel endpoint family
- the single-order cancel endpoint is still unknown

### Pending Order Amend Path

User-driven action:

- a pending TSLL buy order at `1,000원` was opened from `주문내역 > 대기`
- the user clicked `수정` and changed the order price to `500원`

Observed amend result:

- `GET /api/v3/trading/order/4/available-actions?stockCode=US20220809012&tradeType=buy&orderPriceType=00&fractional=false&isReservationOrder=false` -> `200 OK`
- `POST /api/v2/wts/trading/order/correct/prepare/2026-03-11/4` -> `200 OK`
- `POST /api/v2/wts/trading/order/correct/2026-03-11/4` -> `200 OK`
- follow-up reconciliation:
  - repeated `GET /api/v1/trading/orders/histories/PENDING?...`

Observed post-amend state:

- CLI `orders list` surfaced:
  - `orderPrice: 500`
  - `orderUsdPrice: 0.3395`
  - `quantity: 1`
  - `pendingQuantity: 1`
  - `status: 체결대기`
  - `correctSupport: true`
- account summary reflected:
  - `pending_buy_order_amount: 500`
  - `orderable_amount_krw: 500`
  - `orderable_amount_usd: 0.34`

Current inference:

- amend uses its own prepare and mutation pair, parallel to place and bulk cancel
- the pending order remained open after amend and must be reconciled through pending history
- the path key used in `correct/.../4` did not match the later CLI-visible pending `orderNo`, so amend may replace or rekey the order record
- trading implementation should treat amend success as requiring full post-mutation reconciliation rather than trusting the pre-amend identifier

### Captured Amend Request Bodies

For a later amend where pending order `13` was repriced from `600원` to `700원`, the browser emitted:

`correct/prepare` body:

```json
{
  "stockCode": "US20220809012",
  "market": "NSQ",
  "currencyMode": "KRW",
  "tradeType": "buy",
  "price": 0.4753,
  "quantity": 1,
  "orderAmount": 0,
  "orderPriceType": "00",
  "agreedOver100Million": false,
  "max": false,
  "isReservationOrder": false,
  "openPriceSinglePriceYn": false,
  "withOrderKey": true
}
```

`correct` body:

```json
{
  "stockCode": "US20220809012",
  "market": "NSQ",
  "currencyMode": "KRW",
  "tradeType": "buy",
  "price": 0.4753,
  "quantity": 1,
  "orderAmount": 0,
  "orderPriceType": "00",
  "agreedOver100Million": false,
  "max": false,
  "isReservationOrder": false,
  "openPriceSinglePriceYn": false
}
```

Additional observations from the same amend:

- input price in the dialog was `700원`
- pending order after reconciliation became `orderNo: 14`
- resulting pending order still showed:
  - `status: 체결대기`
  - `quantity: 1`
  - `price: 700`

Current inference:

- amend sends USD price precision even while `currencyMode` remains `KRW`
- the browser-side USD conversion matched `700 / 1472.8 = 0.4753`
- `withOrderKey` appears on `correct/prepare` but not on `correct`
- CLI behaviour (`tossctl order place`): `--currency-mode KRW` (default) divides the input by the live exchange rate before sending; `--currency-mode USD` sends the USD value as-is. In both cases the wire payload keeps `currencyMode: "KRW"` to match every captured order/prepare body.

### Pending Order Inline Cancel Path

User-driven action:

- the amended TSLL pending order was opened from `주문내역 > 대기`
- the user clicked the inline `취소` action instead of `전체 취소`

Observed cancel result:

- `GET /api/v3/trading/order/5/available-actions?stockCode=US20220809012&tradeType=buy&orderPriceType=00&fractional=false&isReservationOrder=false` -> `200 OK`
- `POST /api/v2/wts/trading/order/cancel/prepare/2026-03-11/5` -> `200 OK`
- `POST /api/v3/wts/trading/order/cancel/2026-03-11/5` -> `200 OK`
- follow-up reconciliation:
  - repeated `GET /api/v1/trading/orders/histories/PENDING?...`

Observed post-cancel state:

- browser UI showed `대기중인 주문이 없어요`
- CLI `orders list --output json` returned `[]`

Current inference:

- single-order cancel is distinct from bulk cancel and uses its own prepare and mutation pair
- the cancel mutation path key matched the current pending order identity `5`, unlike amend which acted on the previous path key `4`
- initial trading implementation can support both bulk cancel and single-order cancel, but both still require reconciliation through pending history

### Captured Inline Cancel Request Bodies

For a later inline cancel on pending order `14`, the browser emitted:

`cancel/prepare` body:

```json
{
  "isAfterMarketOrder": false,
  "quantity": 1,
  "stockCode": "US20220809012",
  "tradeType": "buy",
  "withOrderKey": true,
  "isReservationOrder": false
}
```

`cancel` body:

```json
{
  "isAfterMarketOrder": false,
  "quantity": 1,
  "stockCode": "US20220809012",
  "tradeType": "buy",
  "isReservationOrder": false
}
```

Additional observations from the same cancel:

- pending row before cancel was:
  - `orderNo: 14`
  - `orderPrice: 700`
  - `pendingQuantity: 1`
- browser showed `구매 주문 취소`
- pending history and CLI both reconciled to zero remaining pending orders

Current inference:

- single-order cancel mirrors amend structure by sending `withOrderKey` only on the prepare step
- cancel uses order metadata from the current pending record rather than only the path key
- `quantity` and `stockCode` are required in both the preflight and final cancel body

## Unknowns

- full response body shape for `order-data`
- whether sell preview also calls `order-data` under a different event boundary
- request body for actual place, cancel, and amend mutations
- request body for `order/prepare`
- request body for `order/create`
- any explicit FX-conversion or funding-approval request that occurs between a failed `prepare` and a successful retry
- request body for `bulk-cancel/prepare`
- request body for `bulk-cancel`
- idempotency or duplicate-submit protection fields
- any hidden nonce or ticket required on live submit
