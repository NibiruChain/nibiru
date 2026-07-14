# Nibiru - /evm

Nibiru EVM: the Go implementation of Nibiru's Ethereum-compatible execution layer. This package holds the Cosmos SDK module, state keeper, ante handlers, transaction types, and Nibiru-specific EVM precompiles.

```bash
⚡ NibiruChain/Nibiru/evm
├── 📂 cli              # `nibid` query and tx helpers for the EVM module
├── 📂 e2e              # JSON-RPC integration tests (ethers.js, Hardhat, Bun)
├── 📂 embeds           # Solidity fixtures and `@nibiruchain/solidity` publish artifacts
├── 📂 evmante          # AnteHandler steps for `MsgEthereumTx`
├── 📂 evmmodule        # Cosmos SDK `AppModule` wiring
├── 📂 evmstate         # EVM keeper, `StateDB` (`SDB`), and message server
├── 📂 evmtest          # Shared Go test helpers for EVM unit and integration tests
├── 📂 forge            # Foundry workspace for Solidity dev and passkey/P-256 tests
├── 📂 precompile       # Nibiru custom precompiles (FunToken, Oracle, Wasm, P-256)
├── *.go                # Core types: txs, genesis, params, FunToken, zero-gas
└── README.md
```

Related paths outside this directory:

- directory `eth/` — Ethereum JSON-RPC, account encoding, EIP-712 signing
- directory `proto/eth/evm/v1/` — protobuf definitions for the EVM module

## Hacking

Install command `just` to run repo-level recipes. From the repository root, run command `just -l` to list them.

### Go unit tests

Run package tests from the repository root:

```bash
go test ./evm/...
```

The directory `evm/evmtest/` exports helpers used across EVM-related tests in this repo.

### EVM end-to-end tests

The directory `evm/e2e/` runs integration tests against a live Nibiru node with JSON-RPC enabled. See file `evm/e2e/README.md` for setup (localnet, `.env`, passkey/ERC-4337 flows).

From the repository root:

```bash
just test-e2e   # requires localnet; runs `just test` in evm/e2e/
```

Or from directory `evm/e2e/`:

```bash
just install
just test
```

### Solidity embeds

Directory `evm/embeds/` holds Hardhat-compiled test contracts and the npm package `@nibiruchain/solidity`. Regenerate ABIs with:

```bash
just gen-embeds
```

See file `evm/embeds/README.md` for npm usage.

### Foundry workspace

Directory `evm/forge/` is a Foundry project for Solidity development, including passkey/P-256 smart-account tests. See file `evm/forge/README.md`.

### Precompiles

Package `evm/precompile` registers Nibiru custom precompiles alongside the standard Berlin set:

| Precompile | Address |
| --- | --- |
| FunToken | `0x0000000000000000000000000000000000000800` |
| Oracle | `0x0000000000000000000000000000000000000801` |
| Wasm | `0x0000000000000000000000000000000000000802` |
| P-256 (RIP-7212) | `0x0000000000000000000000000000000000000100` |

Public docs: [Nibiru EVM precompiles](https://nibiru.fi/docs/evm/precompiles/nibiru.html).

## Zero-gas EVM fee exemption

EVM transactions whose `to` address is listed in `always_zero_gas_contracts` and whose transaction data passes normal non-fee validation are treated as fee-exempt. The raw signed Ethereum transaction is preserved for signature checks, transaction hashes, RPC transaction views, tracing, and debugging. After classification, the derived EVM execution message uses zero fee-price fields, and native NIBI gas fees are not required, deducted, burned, tipped, or refunded. If the transaction attaches native value through field `msg.value`, the sender must still have enough EVM native balance to cover that value.

Operators can recompute eligibility from recorded chain data:

```bash
nibid q sudo zero-gas-actors --height <height>
nibid q tx <tx-hash> --height <height>
```

Decode message type `MsgEthereumTx`, compare its signed `to` field against the allowlist at the transaction height, and confirm that the sender's native NIBI balance did not decrease by an EVM gas payment. A nonzero `value` may still move native NIBI as part of EVM execution. Fee-exempt EVM transactions still meter execution gas and count against block gas limits; event field `EventEthereumTx.gas_used` may be nonzero, while the usual `tx.fee` event from EVM gas deduction is absent.
