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

Starting with CLI v2.23.1, case-insensitive `USDC` selects Sai USDC and
`USDC.e` selects legacy Stargate USDC. The CLI sends explicit token addresses,
so these aliases work against existing v2.23.0 nodes. Use explicit deployed
addresses or bank denoms on networks other than mainnet.

This CLI patch does not change the node's `/eth.evm.v1.Query/Balance` handler.
Direct gRPC and REST requests, including Wasm contract Stargate queries, retain
the v2.23.0 interpretation of `USDC` as legacy USDC. Use explicit addresses or
bank denoms in those interfaces to select either asset. Changing node alias
semantics requires a coordinated chain upgrade and is developed separately.
