# Toss Securities RPC Catalog

Verified from public web traffic and public page navigation on 2026-03-11.

This file is the source of truth for endpoint discovery. It should grow before the Go client grows.

## Status Legend

- `public`: works without login
- `guest`: works before authenticated account state, but may depend on browser bootstrap
- `auth`: requires a logged-in web session
- `blocked`: excluded from CLI scope
- `unknown`: not captured yet

## Hostnames

| Hostname | Role | Notes |
| --- | --- | --- |
| `wts-api.tossinvest.com` | core web runtime and session bootstrap | likely holds login and user-setting paths |
| `wts-info-api.tossinvest.com` | market and UI data | strong candidate for read-only quote and stock detail data |
| `wts-cert-api.tossinvest.com` | certified or sensitive read paths | comments, indicators, some overview widgets |
| `cdn-api.tossinvest.com` | refresh and static coordination | low direct CLI value so far |
| `tuba-static.tossinvest.com` | static variables | not a CLI target |
| `log.tossinvest.com` | telemetry | blocked from CLI scope |

## Bootstrap and Runtime

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `guest` | `GET` | `wts-api.tossinvest.com` | `/api/v3/init?tabId=...` | browser tab bootstrap | `.result` is boolean `true` in public capture | none | useful for reproducing minimal browser session behavior |
| `public` | `GET` | `wts-api.tossinvest.com` | `/api/v1/time` | server time | object under `.result` | none | likely helpful for request signing or freshness checks later |
| `guest` | `GET` | `wts-api.tossinvest.com` | `/api/v1/user-setting` | current user or guest settings | object under `.result` | none | seen without login |
| `public` | `GET` | `wts-api.tossinvest.com` | `/api/v2/system/trading-hours/integrated` | trading-hours metadata | object under `.result` | future metadata | useful for quote context |
| `blocked` | `POST` | `log.tossinvest.com` | `/api/v1/perf-log/bulk` | telemetry | not relevant | none | never call from CLI |
| `blocked` | `POST` | `log.tossinvest.com` | `/api/v2/log/bulk` | telemetry | not relevant | none | never call from CLI |

## Login and Session Discovery

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `guest` | `GET` | `www.tossinvest.com` | `/signin?redirectUrl=%2Faccount` | login page | HTML form with phone and QR flows | `auth login` entry | visiting `/account` without auth redirects here |
| `guest` | `POST` | `wts-api.tossinvest.com` | `/api/v2/login/wts/toss/cert-init` | login flow bootstrap | request body still undocumented | `auth login` helper only | observed both before and after login redirect |
| `guest` | `POST` | `wts-api.tossinvest.com` | `/api/v2/login/wts/toss/qr` | start QR-based login | request body still undocumented | `auth login` helper only | observed in successful QR flow |
| `guest` | `GET` | `wts-api.tossinvest.com` | `/api/v2/login/wts/toss/status` | poll QR login state | object under `.result` | `auth login` helper only | repeated polling until approval |
| `guest` | `POST` | `wts-api.tossinvest.com` | `/api/v2/login/wts/toss` | finalize Toss login | request body still undocumented | `auth login` helper only | observed after status polling |
| `guest` | `POST` | `wts-api.tossinvest.com` | `/api/v3/login/ticket` | obtain post-login ticket | request body still undocumented | `auth login` helper only | likely bridges login flow into WTS session |
| `auth` | `mixed` | browser cookies and storage | session persistence state | authenticated session reuse | cookies plus local/session storage | `auth status`, `auth login` | state-save capture showed both cookies and storage keys matter |

## Market Overview

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/trading-info` | dashboard trading-hours cards | `.result.data[]` with `key`, `name`, `marketOpen`, `currentMarketTradingHour` | none | useful reference data, not first-class CLI target |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/exchange-rates` | exchange-rate summary | object under `.result` | none | may support quote context |
| `public` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/indicator/index?market=kr` | market indicators | `.result.majorIndicatorInfos` | none | public page dependency |
| `public` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/calendar/economic-events` | calendar snippets | object under `.result` | none | public page dependency |
| `public` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v2/dashboard/wts/overview/ranking` | overview ranking widgets | object under `.result` | none | body contract still needs capture |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v1/dashboard/intelligences/all` | dashboard cards | object under `.result` | none | body contract still needs capture |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/dashboard/wts/overview/signals` | signal cards on stock detail/home | object under `.result` | none | body contract still needs capture |

## Quote and Symbol Detail

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/{code}` | symbol metadata | `.result` object with `symbol`, `name`, `market`, `currency`, `isinCode`, `status` | `quote get` | best starting point for product metadata |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-detail/ui/{code}/common` | stock detail UI metadata | `.result` object with `symbol`, `name`, `badges`, `notices`, `memoCount` | `quote get` | likely useful for enriched quote view |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/product/stock-prices?meta=true&productCodes=...` | bulk price lookup | `.result[]` with `productCode`, `base`, `close`, `currency`, `exchange`, `volume` | `quote get`, watchlist | strong candidate for quote batch retrieval |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/c-chart/kr-s/{code}/day:1?...` | chart candles | `.result` with `candles`, `exchange`, `exchangeRate`, `nextDateTime` | later charts | not required for first release |
| `public` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v4/comments?subjectType=STOCK&subjectId=...` | community comments | object under `.result` | none | exclude from first release due to identity and moderation concerns |

## Rankings and Watch Surface

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/rankings/realtime/stock?size=10` | realtime ranking list | object under `.result` | none | useful for future discovery commands |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-infos?codes=...` | bulk metadata lookup | object under `.result` | future watchlist | useful companion to bulk price lookup |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/screener/screen/search/modal` | screener modal data | object under `.result` | none | outside first release scope |

## Account, Portfolio, Orders, Watchlist

These are approved CLI targets. Initial authenticated discovery happened on 2026-03-11 from the `/account` page after QR login.

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v1/account/list` | account list and primary account key | `.result.accountList`, `.result.primaryKey` | `account list` | high-value first endpoint; sanitize account identifiers |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v3/my-assets/summaries/markets/all/overview` | total assets and profit summary | `.result.accountNo`, `totalAssetAmount`, `evaluatedProfitAmount`, `profitRate`, `overviewByMarket` | `account summary`, `portfolio allocation` | account number appears in response |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v1/my-assets/summaries/markets/kr/withdrawable-amount` | KRW withdrawable amounts | `.result.amount0..amount3`, `.result.date0..date3` | `account summary` | public account summary dependency |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v1/my-assets/summaries/markets/us/withdrawable-amount` | USD withdrawable amounts | `.result.amount0..amount3`, `.result.date0..date3` | `account summary` | public account summary dependency |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/orders/histories/all/pending` | pending order history | `.result` list | `orders list` | initial capture returned an empty list |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/common/cached-orderable-amount` | orderable buying power | `.result.orderableAmountKr`, `.result.orderableAmountUs` | `account summary` | useful for summary view |
| `auth` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/asset/sections/all` | account dashboard sections | body `{"types":["MIDDLE"]}` (and others) | dashboard middle banner | filter required since 2026-05-13 (#29) |
| `auth` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v2/dashboard/asset/sections/all` | account dashboard sections v2 | body `{"types":["SORTED_OVERVIEW"\|"WATCHLIST"\|...]}` | `portfolio positions`, `watchlist list` | **2026-05-13: empty `{}` body now returns empty sections + `pollIntervalMillis`. Must pass `types` filter.** |
| `auth` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v1/profit/overview` | profit overview widget | body contract unknown | `portfolio allocation` | body still needs capture |

Watchlist-specific endpoints are still not isolated. The `/account` page did not clearly expose a standalone watchlist read path in the first authenticated capture.

## Transactions Ledger

Captured via `/my-assets` navigation on 2026-04-19. Covers trades, cash flow, dividends, and stock in/out per market.

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v3/my-assets/transactions/markets/{market}` | paginated transaction ledger | `.result.body[]` with `type`, `transactionType.{code,displayName}`, `stockCode`, `stockName`, `quantity`, `amount`, `adjustedAmount`, `commissionAmount`, `totalTaxAmount`, `balanceAmount`, `date`, `dateTime`, `settlementDate`, `referenceType`, `referenceId`, `compositeKey` | `transactions list` | `market` = `kr` or `us`. Query params: `size`, `filters` (0=all, 1=trades, 2=cash/dividend, 3=stock in-out, 6=alt cash; 4/5/7 return 500), `range.from`, `range.to`. `size` is honored; `range.from` and `number` are silently ignored — Toss returns up to `size` entries within the tail of `range.to`. Items are grouped by `type` ASC (1 = trade records, 2 = cash-flow records), then DESC by `dateTime`/`date` inside each group. US `type=1` trades populate only `settlementDate` (T+2); client range-filter falls back to `compositeKey.orderDate` to match execution day. Client pages older data by re-issuing with `range.to` set to the earliest date seen, dedupes by SortKey (derived from `compositeKey`), and filters items to the caller's `[from, to]` window. Max range = 200 days (client-side guard). |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v3/my-assets/transactions/markets/{market}/overview` | cash overview per market | `.result` with `orderableAmount`, `withdrawableAmount.amount0..3`, `depositAmount.amount0..3`, `estimateSettlementAmount.day1..2`, `withdrawableAmountBottomSheet` | `transactions overview` | `depositAmount` buckets represent upcoming settlement credits; `estimateSettlementAmount` shows buy/sell amounts clearing on each upcoming settlement date. |

## Read-Only Policy Notes

The Go client should only admit endpoints that are:

- observed in this catalog
- explicitly classified as read-only
- mapped to an approved CLI command

The following classes stay blocked:

- any order placement endpoint
- any order modification or cancelation endpoint
- any watchlist mutation endpoint unless scope changes
- telemetry endpoints
- comment posting or social actions

## Next Catalog Work

1. Capture authenticated account flows with a clean browser session.
2. Record request bodies for `cert-init`, ranking, and signals endpoints.
3. Promote quote-related endpoints into typed Go client methods.
4. Add stable fixture names for every supported endpoint family.
