# evm

To install dependencies:

```bash
bun install
```

To run:

```bash
bun run README.ts
```

This project was created using `bun init` in bun v1.0.28. [Bun](https://bun.sh) is a fast all-in-one JavaScript runtime.

## Zero-Gas EVM Fee Exemption

EVM transactions whose `to` address is listed in `always_zero_gas_contracts` and
whose `value` is zero are treated as fee-exempt. The raw signed Ethereum
transaction is preserved for signature checks, transaction hashes, RPC
transaction views, tracing, and debugging. After classification, the derived EVM
execution message uses zero fee-price fields, and native NIBI gas fees are not
required, deducted, burned, tipped, or refunded.

Operators can recompute eligibility from recorded chain data:

```bash
nibid q sudo zero-gas-actors --height <height>
nibid q tx <tx-hash> --height <height>
```

Decode the `MsgEthereumTx`, compare its signed `to` and `value` fields against
the allowlist at the transaction height, and confirm that the sender's native
NIBI balance did not decrease by an EVM gas payment. Fee-exempt EVM txs still
meter execution gas and count against block gas limits; `EventEthereumTx.gas_used`
may be nonzero, while the usual `tx.fee` event from EVM gas deduction is absent.
