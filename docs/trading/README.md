# Trading Discovery

This directory is the tracked home for trading-specific reverse-engineering notes.

It is intentionally separate from the local-only planning documents under `docs/plans/`.

Trading execution is not enabled by these documents alone. Live trading actions stay disabled by default until the local `config.json` explicitly allows each action.

See also:

- [`../configuration.md`](../configuration.md)
- [`../../schemas/config.schema.json`](../../schemas/config.schema.json)

Expected contents:

- `rpc-catalog.md`
- `order-state-machine.md`
- `error-codes.md`

Rules:

- do not commit raw storage-state files
- do not commit raw order captures
- do not commit secrets, tokens, or account numbers
- sanitize trading responses before adding them here
