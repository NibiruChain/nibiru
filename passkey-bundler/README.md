# Passkey Bundler

Lightweight ERC-4337 bundler focused on passkey-backed accounts on Nibiru, aligned with `bundler-prd.md`. It exposes
JSON-RPC on port `4337` by default, performs validation and queue-based submission to the configured EntryPoint, and
ships with health and metrics endpoints for operations.

## Features
- JSON-RPC: `eth_chainId`, `eth_supportedEntryPoints`, `eth_sendUserOperation`, `eth_getUserOperationReceipt`.
- Passkey helpers: `passkey_createAccount(qx,qy,factory?)`, `passkey_getLogs(limit)`.
- Validation: entry point and chain ID enforcement, userOp schema checks, rate limiting, optional API key auth
  (`x-api-key` or `Authorization: Bearer <key>`).
- Queue + retries: in-memory FIFO queue (tie-breaker `maxPriorityFeePerGas`), configurable concurrency, retries with
  gas bumping and nonce management.
- Observability: `/healthz`, `/readyz`, `/metrics` (Prometheus), structured logs kept in a rolling buffer.
- Storage: in-memory receipt/log store with configurable retention (intended to be swapped for SQLite/Postgres later).

## Quickstart

```bash
cd passkey-bundler
npm install               # already run in repo; keeps package-lock.json

# run in dev mode (ts)
RPC_URL=http://127.0.0.1:8545 \
ENTRY_POINT=0x... \
BUNDLER_PRIVATE_KEY=0x... \
BUNDLER_PORT=4337 \
CHAIN_ID=9000 \
npm run dev

# or build then run
npm run build
node dist/index.js
```

## Configuration (env)

- `BUNDLER_MODE`: `dev` (default) or `testnet` (enforces safer defaults + requires `DB_URL`).
- `RPC_URL` / `JSON_RPC_ENDPOINT`: upstream RPC endpoint.
- `ENTRY_POINT`: EntryPoint address (required).
- `CHAIN_ID`: chain ID (falls back to RPC if unset).
- `BUNDLER_PRIVATE_KEY`: bundler signer (required). Optional `BENEFICIARY` overrides the handleOps beneficiary.
- `BUNDLER_PORT`: JSON-RPC/health/metrics port (default `4337`).
- `METRICS_PORT`: optional separate port for metrics.
- `DB_URL`: SQLite database location (e.g. `sqlite:./data/bundler.sqlite` or `./data/bundler.sqlite`).
- `MAX_BODY_BYTES`: request body size limit (default `1000000`).
- `BUNDLER_REQUIRE_AUTH`: require API key auth even if `BUNDLER_API_KEYS` is empty (defaults to `true` in testnet mode).
- `MAX_QUEUE` (default `1000`), `QUEUE_CONCURRENCY` (default `4`).
- `RATE_LIMIT`: requests/minute per IP or API key (default `120`); `BUNDLER_API_KEYS` as comma-separated list enables
  auth.
- `GAS_BUMP` (percent, default `15`), `GAS_BUMP_WEI` (absolute bump), `SUBMISSION_TIMEOUT_MS` (default `45000`),
  `FINALITY_BLOCKS` (default `2`).
- `VALIDATION_ENABLED`: run `simulateValidation` before enqueue (defaults to `true` in testnet mode).
- `ENABLE_PASSKEY_HELPERS`: enable `passkey_*` helper RPC methods (defaults to `false` in testnet mode).
- `RECEIPT_LIMIT` (default `1000`), `RECEIPT_POLL_INTERVAL_MS` (default `5000`).

Health: `GET /healthz` (process + RPC reachability), `GET /readyz` (RPC synced, signer nonce). Metrics: `GET /metrics`
Prometheus text.

## Notes
- Queue and receipt storage are in-memory for now; the `BundlerStore` interface is intentionally pluggable for
  SQLite/Postgres backends in a follow-up.
- Rate limiting and API keys are best-effort protections; front a TLS terminator/reverse proxy in production.
