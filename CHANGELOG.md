# Changelog

All notable changes to this project will be documented in this file.

## [0.1.7] - 2026-03-21

### Added
- Korean stock (국내주식) trading support — `tossctl order place --symbol 005930 --market kr`
- `trading.kr` config toggle (default: false) — KR requires `trading.place` and `trading.kr`
- KR branch in `buildPlaceBody`: raw KRW price, no USD conversion, no `allowAutoExchange`
- KR branch in `PlacePendingOrder`: skip USD exchange rate fetch and FX consent
- `InferMarketFromStockCode()` for cancel/amend market recovery from stock code pattern
- KR symbol detection in `NormalizePlace`: numeric 6-digit + market=us → error with guidance
- 13 new test cases (T1-T13) for KR gate, preview, config, payload, reconciliation, symbol detection
- KR cancel/amend live verification TODO

### Changed
- `placeIntentSupported()` now allows both "us" and "kr" markets (was us-only)
- `Place()` reordered: capability → market policy → side policy → execution guard (was guard-first)
- Reconciliation functions parameterized: `Market: "us"` hardcoding (8 places) → market parameter
- TODOS.md: reconciliation parameterization marked as completed
- README, architecture.md, configuration.md updated with KR documentation

## [0.1.6] - 2026-03-21

### Added
- Sell order support for US limit / KRW / non-fractional — `tossctl order place --side sell`
- `trading.sell` config toggle (default: false) — sell requires both `trading.place` and `trading.sell` to be enabled
- Sell policy gate in `Place()` with distinct "disabled by config" error (not "unsupported")
- Sell state reflected in `PreviewPlace` warnings and `MutationReady`
- Sell toggle visible in `config show`, `doctor`, and `EnabledActions()`
- 10 new test cases (T1-T10) covering sell gate, preview, config parsing, payload, and error messages
- `TODOS.md` for tracking deferred work (reconciliation parameterization, sell live verification)

### Changed
- `placeIntentSupported()` no longer restricts by side — both buy and sell are broker-capable
- Warning message updated: "US buy/sell limit orders" (was "US buy limit orders")
- `ErrPlaceUnsupported` message no longer includes `--side buy`
- Doctor trading_scope check updated to reflect buy/sell support
- README, architecture.md, configuration.md updated with sell documentation
