---
order: 7
---

# Querying Prices From CosmWasm

Native module `x/oracle` is deprecated as of v2.12. Its messages are disabled,
validators no longer vote on prices, and the upgrade cleared its stored exchange
rates. A Stargate query to `nibiru.oracle.v1.Query` doesn't return current
prices.

Prices come from the Sai oracle. Wasm contracts read them through the `x-oracle`
adapter contract, which the EVM oracle precompile also uses. For the Sai oracle
contract itself and current integration guides, see the
[Sai developer documentation](https://sai.fun/docs/dev/).

## Adapter Address

The v2.15 upgrade configured the adapter at the same deterministic address on
mainnet and testnet:

```text
nibi1jc6cvkzj73mz685kfdydhugnx4evhn6mupngvhpmw5sapr98mp2sdshh3r
```

Query EVM params and check that plugin `x-oracle` points at this address before
relying on it.

## Adapter Queries

| Query | Returns |
| --- | --- |
| `{"legacy_exchange_rate":{"symbol":"ubtc:uusd"}}` | `symbol`, `token_index`, `price_18`, `price_decimal`, `decimals`, `update_time_seconds` |
| `{"legacy_exchange_rates":{}}` | `rates`, a list of the same objects |
| `{"get_price":{"index":0}}` | `price`, `last_oracle_address`, `last_update_time` for a Sai token index |
| `{"config":{}}` | `mode` and `legacy_mappings` |

Supported legacy symbols are `ubtc:uusd`, `ueth:uusd`, `uatom:uusd`,
`unibi:uusd`, and `uusdc:uusd`. Check `update_time_seconds` before using a
price.

From the CLI:

```bash
nibid query wasm contract-state smart \
  nibi1jc6cvkzj73mz685kfdydhugnx4evhn6mupngvhpmw5sapr98mp2sdshh3r \
  '{"legacy_exchange_rate":{"symbol":"unibi:uusd"}}'
```

From a CosmWasm contract:

```rust
use cosmwasm_schema::cw_serde;
use cosmwasm_std::{Deps, StdResult};

#[cw_serde]
enum AdapterQuery {
    LegacyExchangeRate { symbol: String },
}

#[cw_serde]
struct LegacyExchangeRateResp {
    symbol: String,
    token_index: u16,
    price_18: String,
    price_decimal: String,
    decimals: u8,
    update_time_seconds: Option<u64>,
}

pub fn price(deps: Deps, adapter: &str, symbol: &str) -> StdResult<LegacyExchangeRateResp> {
    deps.querier.query_wasm_smart(
        adapter,
        &AdapterQuery::LegacyExchangeRate { symbol: symbol.to_string() },
    )
}
```

## Related Documentation

- [Nibiru Wasm Integration Guide](./cw-manage.md)
- [Oracle Precompile (EVM)](../../evm/precompiles/oracle.md)
- [Sai developer documentation](https://sai.fun/docs/dev/)
