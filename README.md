# tossinvest-cli

Unofficial CLI for Toss Securities web workflows.

Project name: `tossinvest-cli`  
CLI binary: `tossctl`

## Status

The project is past the initial scaffold stage.

Working today:

- browser-assisted login with reusable local session storage
- read-only account, portfolio, orders, watchlist, and quote commands
- live trading beta for a narrow slice:
  - `US`
  - `buy`
  - `limit`
  - `KRW`
  - `non-fractional`
- live same-day pending order cancel
- trading reverse-engineering docs and sanitized fixtures

Still being hardened:

- `order amend` needs another fresh end-to-end verification after the latest `place/cancel` fixes
- immediate-fill reconciliation is not as strong as pending-order reconciliation
- broader trading coverage such as `sell`, `market`, `KR`, and `fractional` is not done yet

## Architecture

- `Go`: main CLI, domain model, read-only client, trading client, output rendering, session lifecycle
- `Python`: browser login helper and reverse-engineering utilities
- `Rust`: optional later addition for isolated performance-sensitive workers if there is a real need

Tracked references:

- [`docs/reverse-engineering/`](docs/reverse-engineering/)
- [`docs/trading/`](docs/trading/)

## Current Command Surface

```bash
tossctl auth login
tossctl auth status
tossctl auth logout

tossctl account list
tossctl account summary
tossctl portfolio positions
tossctl portfolio allocation
tossctl orders list
tossctl watchlist list
tossctl quote get <symbol>

tossctl order preview
tossctl order place
tossctl order cancel
tossctl order amend
tossctl order permissions status
tossctl order permissions grant --ttl 300
tossctl order permissions revoke

tossctl export positions --format csv
tossctl export orders --format json
```

## Supported Flows

Read-only flows that work now:

- `auth login`
- `auth status`
- `auth logout`
- `quote get`
- `account list`
- `account summary`
- `portfolio positions`
- `portfolio allocation`
- `orders list`
- `watchlist list`

Trading flows that work now:

- `order preview`
- `order place` for `US buy limit / KRW / non-fractional`
- `order cancel` for same-day pending orders

Trading flows that exist but should still be treated as beta:

- `order amend`

## Trading Safety Model

Trading is intentionally awkward to execute.

Mutation commands require:

- `order preview` to generate the canonical intent and confirm token
- `order permissions grant --ttl 300`
- `--execute`
- `--dangerously-skip-permissions`
- `--confirm <token>`

This is deliberate. The CLI is not trying to optimize accidental order placement.

## Local Paths

By default, the CLI uses OS-native paths:

- config dir: `$(os.UserConfigDir)/tossctl`
- cache dir: `$(os.UserCacheDir)/tossctl`
- session file: `<config dir>/session.json`
- permission file: `<config dir>/trading-permission.json`

During development you can override paths with:

- `--config-dir`
- `--session-file`

## Development

```bash
make tidy
make fmt
make build
make test
cd auth-helper && python3 -m pip install -e . && python3 -m playwright install chromium
./bin/tossctl --help
./bin/tossctl auth login
./bin/tossctl auth status
./bin/tossctl quote get TSLL --output json
./bin/tossctl account summary --output json
./bin/tossctl orders list --output json
```

## Example Trading Flow

```bash
./bin/tossctl order preview \
  --symbol TSLL \
  --market us \
  --side buy \
  --type limit \
  --qty 1 \
  --price 500 \
  --currency-mode KRW \
  --output json

./bin/tossctl order permissions grant --ttl 300

./bin/tossctl order place \
  --symbol TSLL \
  --market us \
  --side buy \
  --type limit \
  --qty 1 \
  --price 500 \
  --currency-mode KRW \
  --execute \
  --dangerously-skip-permissions \
  --confirm <preview-token> \
  --output json

./bin/tossctl orders list --output json
```

## Warning

This project is unofficial and not affiliated with Toss Securities.

Internal web APIs can change without notice, trading flows can require additional browser-side checks, and mistakes can affect a real account. Use it only if you understand those risks.
