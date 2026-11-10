---
order: 3
title: "Nibiru EVM Token Contracts"
---

# EVM Token Contracts

| Token  | Contract Address | Nibiscan Link |
| ---  | --- | --- |
| WNIBI  | 0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97 | [View Contract](https://nibiscan.io/token/0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97/contract/code?type=erc20) |
| Sai USDC | 0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190 | [View Contract](https://nibiscan.io/token/0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190/contract/code?type=erc20) |
| Legacy USDC.e | 0x0829F361A05D993d5CEb035cA6DF3446b060970b | [View Contract](https://nibiscan.io/token/0x0829F361A05D993d5CEb035cA6DF3446b060970b/contract/code?type=erc20)    |
| WETH   | 0xcdA5b77E2E2268D9E09c874c1b9A4c3F07b37555 | [View Contract](https://nibiscan.io/token/0xcdA5b77E2E2268D9E09c874c1b9A4c3F07b37555/contract/code?type=erc20)    |
| stNIBI | 0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b | [View Contract](https://nibiscan.io/token/0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b/contract/code?type=erc20)    |

## Query USDC Balances

Sai USDC and legacy LayerZero Stargate USDC.e are separate tokens. Both use
six decimals. Sai USDC's bank denom is `erc20/0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190`.

```bash
nibiru q evm balance "$ADDR" 0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190
nibiru q evm balance "$ADDR" 0x0829F361A05D993d5CEb035cA6DF3446b060970b
```

The response reports Bank and ERC20 holdings separately when available. A zero
balance applies only to the selected token. Use `nibiru q bank balances "$ADDR"`
to inspect all bank holdings.

The unreleased CLI resolves case-insensitive `USDC` to Sai USDC and `USDC.e`
to legacy Stargate USDC, sending explicit addresses even to v2.23.0 nodes.
Use explicit deployed addresses or bank denoms on networks other than mainnet.

The corresponding node alias change is state-machine breaking because Wasm
contracts can call this handler through Stargate during transaction execution.
It requires a coordinated chain upgrade. After activation, contract and external
queries use the same aliases. Before activation, direct gRPC and REST requests
to existing nodes still resolve `USDC` to the legacy token; use explicit
addresses to select either asset without relying on aliases.
