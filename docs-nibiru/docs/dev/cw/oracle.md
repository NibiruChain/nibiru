---
order: 7
---

# Querying Oracle Data

## Introduction

This guide provides a step-by-step approach to querying exchange rate data from the Nibiru oracle module using both the Nibiru CLI (`nibid`) and Rust-based CosmWasm smart contracts.

---

## Querying Oracle Data via Nibiru CLI

### 1. Query Exchange Rates

To retrieve the latest exchange rates, run the following command:

```bash
nibid q oracle exchange-rates | jq
```

#### Sample Response

```json
{
  "exchange_rates": [
    { "pair": "uatom:uusd", "exchange_rate": "4.932000000000000000" },
    { "pair": "ubtc:uusd", "exchange_rate": "96897.000000000000000000" },
    { "pair": "ueth:uusd", "exchange_rate": "2674.400000000000000000" },
    { "pair": "uusdc:uusd", "exchange_rate": "1.000100000000000000" },
    { "pair": "uusdt:uusd", "exchange_rate": "0.999900000000000000" }
  ]
}
```

### 2. Query Active Pairs

To get the list of active oracle pairs:

```bash
nibid q oracle actives
```

#### Sample Response

```json
{"actives":["uatom:uusd","ubtc:uusd","ueth:uusd","uusdc:uusd","uusdt:uusd"]}
```

---

## Querying Oracle Data in Rust (CosmWasm Contracts)

### Smart Contract Query Implementation

The following Rust contract implements a query function to fetch exchange rate data using Nibiru's Stargate query mechanism.

### Query Function

```rust
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    use QueryMsg::*;

    match msg {
        GetPrice { pair } => to_json_binary(&query::get_price(deps, pair)?),
    }
}
```

### Query Module

```rust
mod query {
    use super::*;
    use nibiru_std::proto::{nibiru::oracle::QueryExchangeRateRequest, NibiruStargateQuery};
    use crate::msgs::{GetCountResp, GetPriceResp};

    pub fn get_price(deps: Deps, pair: String) -> StdResult<GetPriceResp> {
        let query_request = QueryExchangeRateRequest { pair: pair.clone() };
        let query = query_request.into_stargate_query()?;
        let response: GetPriceResp = deps.querier.query(&query)?;
        Ok(response)
    }
}
```

### Query Message Definitions

```rust
use cosmwasm_schema::{cw_serde, QueryResponses};
cw_ownable::cw_ownable_query;

#[cw_ownable_query]
#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(GetPriceResp)]
    GetPrice { pair: String },
}

#[cw_serde]
pub struct GetPriceResp {
    pub exchange_rate: String,
}
```

---

## Understanding the `nibiru_std` Cargo Package

The `nibiru_std` crate is a utility library for interacting with the Nibiru blockchain in CosmWasm smart contracts. It provides essential types and helper functions to facilitate queries and transactions. In the context of querying oracle data, we use:

- **`nibiru_std::proto`**: Contains protocol buffer definitions for interacting with Nibiru modules.
- **`nibiru::oracle::QueryExchangeRateRequest`**: Represents a request to query exchange rates from the oracle module.
- **`NibiruStargateQuery`**: A trait used for converting messages into Stargate-compatible queries, allowing CosmWasm contracts to interact with the Nibiru chain’s modules.

By leveraging these utilities, smart contracts can efficiently fetch real-time exchange rate data from the blockchain without implementing complex serialization logic.

---

## Related Documentation

For further exploration of Nibiru smart contracts and integrations, refer to:

- [Nibiru Wasm Integration Guide](./cw-manage.md)
- [Nibiru CLI](../cli/README.md)

These resources provide more in-depth explanations and examples on interacting with Nibiru's blockchain using both CLI and smart contracts.
