# Changelog

All notable changes to this project will be documented in this file.

## [0.3.0] - 2026-03-21

### Added
- **Fractional (소수점) order support** — `tossctl order place --symbol TSLL --fractional --amount 18000`
  - US market only, market orders (시장가), amount-based (금액 기반)
  - `trading.fractional` config toggle (default: false)
  - `--amount` flag for specifying KRW amount
  - `--fractional` flag auto-selects market order type
- Fractional policy gate in `Place()` with "disabled by config" error
- `buildPlaceBody` fractional branch: `price=0, quantity=0, orderAmount=<KRW>, orderPriceType=01, isFractionalOrder=true`
- `placeIntentSupported()` now accepts fractional orders (US + market only)
- `NormalizePlace` validates fractional constraints (US only, amount required, auto market order)
- 10 new tests: fractional capability, policy, preview, payload, orderintent validation
- API compatibility verified via prepare dry-run (422 = payload accepted, insufficient balance)

## [0.2.3] - 2026-03-21

### Removed
- **MCP server** (`tossctl-mcp`) — CLI 자체가 AI 에이전트에서 직접 실행 가능하므로 불필요한 추상화 제거
- `make build-mcp` Makefile 타겟
- Release workflow에서 tossctl-mcp 바이너리

## [0.2.2] - 2026-03-21

### Added
- `tossctl quote batch <symbol> [symbol...]` — fetch multiple stock quotes at once
- `tossctl export positions --market kr|us|all` — filter exported positions by market
- `tossctl export orders --market kr|us|all` — filter exported orders by market
- Quote output tests (6 test cases)

### Fixed
- Floating point display artifacts in quote batch table output (e.g., `-0.8500000000000014` → `-0.85`)

## [0.2.1] - 2026-03-21

### Added
- MCP server unit tests (10 test cases) covering initialize, tools/list, tool calls, error handling
- Refactored MCP server to testable pure functions (handleMethod, buildInitializeResponse, buildToolsList)

### Removed
- Unused `stub.go` command helper (export commands now fully implemented)

## [0.2.0] - 2026-03-21

### Added
- **MCP server** (`tossctl-mcp`) — read-only Model Context Protocol server for AI agent integration
  - `get_portfolio_positions` — 보유 포지션 조회
  - `get_account_summary` — 계좌 요약 조회
  - `get_quote` — 종목 시세 조회 (US/KR)
  - `list_pending_orders` — 미체결 주문 조회
  - `list_completed_orders` — 체결 완료 내역 조회 (market filter 지원)
  - `list_watchlist` — 관심 종목 조회
- `tossctl export positions` — CSV 포지션 내보내기 (stub에서 실제 구현으로 전환)
- `tossctl export orders` — CSV 체결 내역 내보내기 (stub에서 실제 구현으로 전환)
- `make build-mcp` Makefile 타겟
- Release workflow에 `tossctl-mcp` 바이너리 포함

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
