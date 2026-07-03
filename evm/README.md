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
whose transaction data passes normal non-fee validation are treated as
fee-exempt. The raw signed Ethereum transaction is preserved for signature
checks, transaction hashes, RPC transaction views, tracing, and debugging. After
classification, the derived EVM execution message uses zero fee-price fields,
and native NIBI gas fees are not required, deducted, burned, tipped, or
refunded. If the transaction attaches native value through `msg.value`, the
sender must still have enough EVM native balance to cover that value.

Operators can recompute eligibility from recorded chain data:

```bash
nibid q sudo zero-gas-actors --height <height>
nibid q tx <tx-hash> --height <height>
```

Decode the `MsgEthereumTx`, compare its signed `to` field against the allowlist
at the transaction height, and confirm that the sender's native NIBI balance did
not decrease by an EVM gas payment. A nonzero `value` may still move native NIBI
as part of EVM execution. Fee-exempt EVM txs still meter execution gas and count
against block gas limits; `EventEthereumTx.gas_used` may be nonzero, while the
usual `tx.fee` event from EVM gas deduction is absent.
