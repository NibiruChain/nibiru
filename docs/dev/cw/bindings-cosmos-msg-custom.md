# CosmosMsg::Custom vs Stargate on Nibiru

## Summary

Nibiru CosmWasm contracts should call Nibiru modules with protobuf-backed
Stargate messages and queries. In contract code, this means using message
variant `CosmosMsg::Stargate`, query variant `QueryRequest::Stargate`, and the
generated Rust types exposed by crate `nibiru-std`.

An earlier Nibiru design explored chain-specific custom bindings through Rust
enums such as enum `NibiruMsg` and enum `NibiruQuery`. That design would have
required Go code in the chain to interpret those custom payloads and convert
them into module calls. Nibiru moved away from that pattern because the module
protobuf definitions are already the shared API used by wallets, the command
line, clients, and contract code.

For implementation examples, see the [Nibiru Rust SDK](./rust-sdk.md), the
[oracle Stargate query guide](./oracle.md), and the
[token factory contract guide](./tf.md).

## How the integration patterns differ

The custom-binding pattern sends chain-specific JSON payloads from a contract to
the host chain. The chain must provide a custom Go interpreter for that JSON API.

Old Custom pattern:

```text
CosmWasm contract
  -> CosmosMsg::Custom with NibiruMsg JSON
  -> Go CustomEncoder
  -> SDK module handler
```

The Stargate pattern sends protobuf bytes through the standard CosmWasm Stargate
message and query variants. The chain routes the decoded protobuf message
through the standard SDK module path.

Current Stargate pattern:

```text
CosmWasm contract
  -> CosmosMsg::Stargate with protobuf bytes
  -> wasmd Stargate encoder
  -> sdk.Msg validation and routing
  -> SDK module handler
```

The same distinction applies to queries. A custom-binding design uses a custom
query enum and trait `CustomQuery`. The current Nibiru pattern uses protobuf
request types with query variant `QueryRequest::Stargate`, while the chain
decides which query paths contracts may call.

## Why Nibiru uses Stargate for module calls

Custom bindings remain an official `wasmd` extension point. They are useful when
a chain needs contract-specific behavior, deterministic query shaping, or
functionality that does not map cleanly to a standard transaction message. The
Nibiru decision is narrower: for module transactions and module queries, use
protobuf-backed messages and standard SDK routing instead of maintaining a
second Nibiru-specific Rust and Go interface.

Osmosis reached a similar conclusion while evaluating custom contract bindings.
In [osmosis-labs/osmosis PR #1484](https://github.com/osmosis-labs/osmosis/pull/1484#issuecomment-1176960176),
ValarDragon wrote:

> Closing for now, as we want to go with an approach of using StargateMsg and
> StargateQuery. This is because we don't want to maintain a separate interface
> layer in go just for cosmwasm -- we should just promise interface compatibility
> at one point, or make a native backwards compatible abstraction.

That quote names the maintenance problem. Custom bindings create another API
surface: Rust message enums, Rust query enums, Go encoders, Go queriers, test
fixtures, mocks, and docs that must stay aligned with the module API. When
protobuf messages are already the module API, crate `nibiru-std` should expose
those generated types to contracts.

## What to use on Nibiru

Use crate `nibiru-std` for generated protobuf types and convenience traits:

- Trait `NibiruStargateMsg` converts generated protobuf messages into message
  variant `CosmosMsg::Stargate`.
- Trait `NibiruStargateQuery` converts generated protobuf request types into
  query variant `QueryRequest::Stargate`.
- The chain-side package
  [`app/wasmext`](https://github.com/NibiruChain/nibiru/tree/master/app/wasmext)
  defines the Stargate query behavior exposed to contracts.

The [`nibi-stargate` contract](https://github.com/NibiruChain/nibiru-wasm/tree/main/contracts/nibi-stargate)
shows how a contract can construct token factory module messages. The
[token factory guide](./tf.md) walks through deploying and using that contract.
The [oracle guide](./oracle.md) shows a Stargate query that constructs protobuf
request type `QueryExchangeRateRequest` and converts it with trait
`NibiruStargateQuery`.

## Historical reference: custom binding implementation pieces

The following notes describe the custom-binding design Nibiru evaluated before
standardizing on Stargate for module calls. They are useful when comparing
Nibiru with chains that still expose chain-specific bindings.

### Rust message enum

A custom-binding implementation typically defines a Rust message enum with one
variant per module action. The field names usually mirror the RPC methods from
the module's protobuf `Msg` service.

For example, a protobuf service might define methods like this:

```proto
service Msg {
  rpc RegisterInterchainAccount(MsgRegisterInterchainAccount)
      returns (MsgRegisterInterchainAccountResponse) {};
  rpc SubmitTx(MsgSubmitTx) returns (MsgSubmitTxResponse) {};
}
```

The protobuf method `RegisterInterchainAccount` would correspond to a custom
message enum variant such as:

```rust
pub enum NibiruMsg {
    /// RegisterInterchainAccount registers an interchain account on a remote chain.
    RegisterInterchainAccount {
        /// connection_id is an IBC connection identifier between chains.
        connection_id: String,

        /// interchain_account_id identifies the contract's remote account.
        interchain_account_id: String,
    },
}
```

That enum would then need Go-side handling that converts each custom variant
into the corresponding SDK message.

### Rust query enum

A custom-binding implementation usually defines a Rust query enum with one
variant per supported module query. Those variants often mirror the RPC methods
from the module's protobuf `Query` service.

For example, Osmosis exposes a `SpotPrice` query in file
`osmosis/gamm/v2/query.proto`:

```proto
service Query {
  rpc SpotPrice(SpotPriceRequest) returns (SpotPriceResponse) {
    option (google.api.http).get =
        "/osmosis/poolmanager/pools/{pool_id}/prices";
  }
}

message SpotPriceRequest {
  uint64 pool_id = 1 [ (gogoproto.moretags) = "yaml:\"pool_id\"" ];
  string base_asset_denom = 2
      [ (gogoproto.moretags) = "yaml:\"base_asset_denom\"" ];
  string quote_asset_denom = 3
      [ (gogoproto.moretags) = "yaml:\"quote_asset_denom\"" ];
}

message SpotPriceResponse {
  string spot_price = 1 [ (gogoproto.moretags) = "yaml:\"spot_price\"" ];
}
```

A bindings crate can wrap those protobuf shapes with contract-facing Rust
types:

```rust
#[cw_serde]
pub struct SpotPriceResponse {
    pub price: Decimal,
}

#[derive(Serialize, Deserialize, Clone, Eq, PartialEq, JsonSchema, Debug)]
pub struct Swap {
    pub pool_id: u64,
    pub denom_in: String,
    pub denom_out: String,
}
```

The custom query enum then references the response type:

```rust
pub enum OsmosisQuery {
    #[returns(SpotPriceResponse)]
    SpotPrice { swap: Swap, with_swap_fee: bool },
}
```

The enum must implement trait `cosmwasm_std::CustomQuery` so CosmWasm can use it
as the custom query type:

```rust
impl CustomQuery for OsmosisQuery {}
```

### Go chain integration

The Rust enum is only half of a custom-binding implementation. The chain also
needs Go code that understands the custom JSON payloads emitted by contracts.
That Go code must convert each custom message and query into the appropriate SDK
module call or query response.

This is the layer Nibiru avoids by using protobuf messages directly. With
Stargate, the generated Rust types in crate `nibiru-std` and the chain's
protobuf definitions stay aligned with the same module API.

## Custom bindings on other chains

The following projects are useful references for chains that expose custom
bindings:

- [Neutron SDK bindings](https://github.com/neutron-org/neutron-sdk/tree/4a5fc14e8725ed3fb530e9b97a41abc3cb1e2278/packages/neutron-sdk/src/bindings)
- [Osmosis bindings](https://github.com/osmosis-labs/bindings/tree/v0.7.0/packages/bindings/src)
- [Terra CosmWasm bindings](https://github.com/terra-money/terra-cosmwasm)
- [Cudos CosmWasm bindings](https://github.com/CudoVentures/cudos-cosmwasm-bindings/tree/21875435ef3ff985b0e54832e70d50b1af72b6a0/packages/cudos-cosmwasm/src)
- [Neutron Go wasm bindings](https://github.com/neutron-org/neutron/tree/v0.3.1/wasmbinding)
- [Osmosis Go wasm bindings](https://github.com/osmosis-labs/osmosis/tree/v15.0.0/wasmbinding)
- [Osmosis protobuf definitions](https://github.com/osmosis-labs/osmosis/tree/v15.0.0/proto/osmosis)
- [Neutron protobuf definitions](https://github.com/neutron-org/neutron/tree/v0.3.1/proto)

## References

- [Nibiru Rust SDK](./rust-sdk.md)
- [Querying oracle data](./oracle.md)
- [Creating fungible tokens guide](./tf.md)
- [`nibi-stargate` contract](https://github.com/NibiruChain/nibiru-wasm/tree/main/contracts/nibi-stargate)
- [`app/wasmext` package](https://github.com/NibiruChain/nibiru/tree/master/app/wasmext)
