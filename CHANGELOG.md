# Changelog

All notable changes to this project will be documented in this file.

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
