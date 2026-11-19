---
order: 5
description: >-
  Read Sai oracle prices from an EVM contract, a CosmWasm contract, a terminal,
  or an AI agent. Every example here was run read-only against mainnet.
---

# Sai Oracle Data and How to Read It

{{ $frontmatter.description }}

Prices on Nibiru come from the Sai oracle, a CosmWasm contract on Sai L1. Two
paths expose it to other contracts: the `x-oracle` adapter contract for Wasm, and
the precompile at `0x0801` for the EVM. Both read the same prices.

## What Data Exists

| Symbol | Sai token index | Precompile pair |
| --- | --- | --- |
| `usdc` | 1 | `uusdc:uusd` |
| `btc` | 3 | `ubtc:uusd` |
| `eth` | 4 | `ueth:uusd` |
| `atom` | 5 | `uatom:uusd` |
| `nibi` | 49 | `unibi:uusd` |

- Prices are in USD.
- The adapter and precompile return prices with 18 decimals. The Sai oracle
  contract returns a decimal string.
- Every response carries the oracle's last update time in Unix seconds. On
  2026-11-19, each query ran within 30 seconds of the reported update time.
  Check the update time before you trust a price.
- Other pairs fail. The precompile returns `unsupported legacy symbol: usdt` for
  `uusdt:uusd`.
- Sai's own markets use more token indexes than the table above (for example
  `stnibi` is 2). See the
  [Sai developer documentation](https://sai.fun/docs/dev/) for the full list.

| Contract | Address |
| --- | --- |
| Sai oracle | `nibi1xfwyfwtdame6645lgcs4xvf4u0hpsuvxrcelfwtztu0pv7n4l6hqw5a8gj` |
| `x-oracle` adapter | `nibi1jc6cvkzj73mz685kfdydhugnx4evhn6mupngvhpmw5sapr98mp2sdshh3r` |

The adapter address is stored in the EVM params field `wasm_plugins` under name
`x-oracle`. You can read it from the REST endpoint:

```bash
curl -s https://lcd.nibiru.fi/nibiru/evm/v1/params
```

```json
{"params":{"...":"...","wasm_plugins":[{"name":"x-oracle","addr":"nibi1jc6cvkzj73mz685kfdydhugnx4evhn6mupngvhpmw5sapr98mp2sdshh3r"}]}}
```

## Read From an EVM Contract

The precompile at `0x0000000000000000000000000000000000000801` is read-only and
has two methods. Both take a pair such as `ubtc:uusd`.

```solidity
interface IOracle {
    function queryExchangeRate(string memory pair) external view returns (
        uint256 price, uint64 blockTimeMs, uint64 blockHeight
    );
    function chainLinkLatestRoundData(string memory pair) external view returns (
        uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound
    );
}

contract PriceReader {
    IOracle constant ORACLE = IOracle(0x0000000000000000000000000000000000000801);

    function btcUsd() external view returns (uint256 price18, uint256 updatedAt) {
        (, int256 answer,, uint256 updated,) = ORACLE.chainLinkLatestRoundData("ubtc:uusd");
        return (uint256(answer), updated);
    }
}
```

| Method | Return fields |
| --- | --- |
| `queryExchangeRate` | `price` with 18 decimals. `blockTimeMs` is the oracle's update time in milliseconds. `blockHeight` is the height of your query. |
| `chainLinkLatestRoundData` | `roundId` is the height of your query. `answer` has 18 decimals. `startedAt` and `updatedAt` are both the oracle's update time in seconds. `answeredInRound` is always 420. |

From a terminal with Foundry:

```bash
cast call 0x0000000000000000000000000000000000000801 \
  "queryExchangeRate(string)(uint256,uint64,uint64)" "ubtc:uusd" \
  --rpc-url https://evm-rpc.nibiru.fi
```

```text
63306010000000002000000 [6.33e22]
1795102232000 [1.795e12]
49952841 [4.995e7]
```

```bash
cast call 0x0000000000000000000000000000000000000801 \
  "chainLinkLatestRoundData(string)(uint80,int256,uint256,uint256,uint80)" "ubtc:uusd" \
  --rpc-url https://evm-rpc.nibiru.fi
```

```text
49953018 [4.995e7]
63175660000000003000000 [6.317e22]
1795102402 [1.795e9]
1795102402 [1.795e9]
420
```

The price `63175660000000003000000` is 63,175.66 USD. The trailing digits come from
the oracle's floating-point price, so round before you display it.

Deployed Chainlink-style feed contracts that wrap this precompile are listed on
the [Chainlink-like feeds page](./evm/oracle.md).

## Read From a CosmWasm Contract

Send a smart query to the adapter. Responses are wrapped in `{"data": ...}`
by the CLI and returned bare to a contract.

| Query | Returns |
| --- | --- |
| `{"legacy_exchange_rate":{"symbol":"ubtc:uusd"}}` | `symbol`, `token_index`, `price_18`, `price_decimal`, `decimals`, `update_time_seconds` |
| `{"legacy_exchange_rates":{}}` | `rates`, a list of the same objects |
| `{"get_price":{"index":3}}` | `price`, `last_oracle_address`, `last_update_time` for a Sai token index |
| `{"config":{}}` | `mode` (the Sai oracle address) and `legacy_mappings` |

```rust
use cosmwasm_schema::cw_serde;
use cosmwasm_std::{Deps, StdResult};

#[cw_serde]
enum AdapterQuery {
    LegacyExchangeRate { symbol: String },
}

#[cw_serde]
pub struct LegacyExchangeRateResp {
    pub symbol: String,
    pub token_index: u16,
    pub price_18: String,
    pub price_decimal: String,
    pub decimals: u8,
    pub update_time_seconds: Option<u64>,
}

pub fn price(deps: Deps, adapter: &str, symbol: &str) -> StdResult<LegacyExchangeRateResp> {
    deps.querier.query_wasm_smart(
        adapter,
        &AdapterQuery::LegacyExchangeRate { symbol: symbol.to_string() },
    )
}
```

The Rust snippet was not compiled. Its query message and response fields match
the CLI output in the next section.

## Read From a Terminal or Script

Wasm, through the adapter:

```bash
nibiru q wasm contract-state smart \
  nibi1jc6cvkzj73mz685kfdydhugnx4evhn6mupngvhpmw5sapr98mp2sdshh3r \
  '{"legacy_exchange_rate":{"symbol":"ubtc:uusd"}}'
```

```json
{"data":{"symbol":"btc","token_index":3,"price_18":"63467500000000000000000","price_decimal":"63467.5","decimals":18,"update_time_seconds":1795102115}}
```

Wasm, from the Sai oracle directly, by token index:

```bash
nibiru q wasm contract-state smart \
  nibi1xfwyfwtdame6645lgcs4xvf4u0hpsuvxrcelfwtztu0pv7n4l6hqw5a8gj \
  '{"get_price":{"index":3}}'
```

```json
{"data":{"price":"63487.300000000003","last_oracle_address":"nibi1nqg2k8pzpsxgn833vax86gn6h7yxp23d8psh94","last_update_time":1795102110}}
```

All adapter prices at once:

```bash
nibiru q wasm contract-state smart \
  nibi1jc6cvkzj73mz685kfdydhugnx4evhn6mupngvhpmw5sapr98mp2sdshh3r \
  '{"legacy_exchange_rates":{}}'
```

```json
{"data":{"rates":[{"symbol":"atom","token_index":5,"price_decimal":"1.285","update_time_seconds":1795102427},{"symbol":"btc","token_index":3,"price_decimal":"63160","update_time_seconds":1795102427},{"symbol":"eth","token_index":4,"price_decimal":"1780.67","update_time_seconds":1795102427},{"symbol":"nibi","token_index":49,"price_decimal":"0.0003721","update_time_seconds":1795102427},{"symbol":"usdc","token_index":1,"price_decimal":"1.00091","update_time_seconds":1795102427}]}}
```

The output above drops fields `price_18` and `decimals` for length.

EVM, with the `cast` commands in the EVM section above.

## For AI Agents

The open-source [`sai` CLI](https://github.com/SaiDotFun/open-sai) prints Sai's
indexed prices for every Sai token, and its MCP server exposes the same read as
tool `market_prices`. Install it with [uv](https://docs.astral.sh/uv/):

```bash
uv run https://raw.githubusercontent.com/SaiDotFun/open-sai/main/scripts/install.py
sai --json market prices
```

```json
{"oracle":{"tokenPricesUsd":[{"priceUsd":63450.4,"token":{"id":3,"symbol":"BTC","name":"btc"},"lastUpdatedBlock":{"block":49952614,"block_ts":"2026-11-19T15:26:54.087377Z"}}]}}
```

The real output lists every token, trimmed here to BTC. Field `lastUpdatedBlock`
shows how fresh each price is. Prices here come from Sai Keeper's index, not
from a chain query, so they may lag the contract.

To register the MCP server with an agent client:

```bash
sai mcp install claude   # or codex, cursor, --all
```

Agents can also run `sai skill show` to load the packaged agent skill.
