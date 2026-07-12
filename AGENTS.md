# AGENTS.md

This file provides practical guidance for working in the Nibiru repository.

## Project Overview

Nibiru Chain is an L1 blockchain with parallel VM support:
- **Nibiru EVM** for Ethereum-compatible execution
- **Wasm VM** for CosmWasm smart contracts

Important Directories:
- `app/` - Application wiring and integration
- `cmd/nibid/` - Node binary entrypoints and localnet scripts
- `evm/` - EVM module, EVM state, ante flow, precompiles, E2E, and Forge workspace
- `eth/` - Ethereum RPC/account/encoding helpers
- `gosdk/` - Go SDK client package
- `lib/cosmos-sdk/` - Nibiru's in-tree Cosmos SDK packages
- `lib/ibc-go/` - Nibiru's in-tree IBC packages
- `proto/` - Protobuf definitions
- `x/` - Custom Cosmos SDK modules

## Core Commands

Prefer `just` commands at the repository root.

### Build and Run

```bash
just install   # build + install nibid
just build     # build nibid without install
just localnet  # start a local chain
```

### Test

```bash
just test-fast     # go test with cache
just test          # go test -count=1 (requires localnet)
just test-cover    # go test with coverage (requires localnet)
just test-e2e      # EVM E2E test suite in evm/e2e/
go test ./evm/...  # target one package tree
```

### Go example tests

Functions in `*_test.go` files whose names begin with `Example` are executable
documentation. The name after `Example` must identify the exported package
symbol being documented:

```go
Example()                 // package
ExampleFunction()         // exported function or type
ExampleType_Method()      // exported method
Example_suffix()          // package-level example with a lowercase suffix
```

An `// Output:` block makes the example's standard output an assertion. Use a
lowercase suffix when an example demonstrates several APIs and does not
document one specific exported symbol.

### Lint, Format, and Proto

```bash
just tidy        # go mod tidy + proto gen + lint + fmt
just proto gen
just lint        # golangci-lint
just fmt         # gofumpt
```

## Environment and Requirements

- Go version: follow `go.mod`
- Node.js and npm: required for `evm/e2e/`
- Docker: required for containerized workflows (for example, some lint/chaosnet flows)

## Common Workflow

1. Install or update dependencies with command `just install`.
2. Run a local chain with command `just localnet` when tests require live chain state.
3. Make code changes in the relevant package directories.
4. Run relevant tests (`go test ./<pkg>/...`, command `just test-fast`, and/or command `just test-e2e`).
5. Run command `just tidy` before opening a PR.

## Notes for EVM Work

- E2E tests live in directory `evm/e2e/` and are run via command `just test-e2e`.
- Solidity embed artifacts are generated with command `just gen-embeds`.
- Foundry workspace is in directory `evm/forge/`.

## Local Endpoints

- RPC: `http://localhost:26657`
- gRPC: `localhost:9090`
- EVM RPC: `http://localhost:8545`
- API: `http://localhost:1317`

## References

- Main docs: https://nibiru.fi/docs/
- TypeScript SDK: https://www.npmjs.com/package/@nibiruchain/ts-sdk
- Solidity package: file `evm/embeds/README.md`
- EVM package: file `evm/README.md`

## Community

- Discord: https://discord.gg/nibirufi
- X (Twitter): https://twitter.com/NibiruChain
- Telegram: https://t.me/nibiruchainofficial
