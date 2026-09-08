# nibiru-wasm/contracts/nibi-stargate

This smart contract showcases usage examples for creating and managing fungible tokens native to Nibiru.

Table of Contents

- [Guide: Using the Smart Contract](#guide-using-the-smart-contract)
  - [Set environmnent vars](#set-environmnent-vars)
  - [Instantiate the Smart Contract](#instantiate-the-smart-contract)
- [Broadcasting Transactions to the Network](#broadcasting-transactions-to-the-network)
  - [Examples](#examples)
- [Appendix: Stargate Types](#appendix-stargate-types)
  - [cosmwasm-std: CosmosMsg::Stargate](#cosmwasm-std-cosmosmsgstargate)
  - [wasmvm: StargateMsg](#wasmvm-stargatemsg)
  - [cosmwasm-std: QueryRequest::Stargate](#cosmwasm-std-queryrequeststargate)
  - [wasmvm: StargateQuery](#wasmvm-stargatequery)

## Guide: Using the Smart Contract

A pre-built version of the Wasm bytecode for every smart contract in the
[NibiruChain/nibiru-wasm](https://github.com/NibiruChain/nibiru-wasm) repo can be
found in the "artifacts" directory.

### Set environmnent vars

This guide assumes you are working with the `nibid` command-line interface. See "[https://nibiru.fi/docs/dev/cli/nibid-binary.html | Nibiru Docs](https://nibiru.fi/docs/dev/cli)" for installation instructions. Note that `nibid` currently only supports Unix systems like MacOS and Linux. We don't yet support vanilla Windows, but you can WSL Ubuntu.

| Name | Description |
| --- | --- |
| `KEYNAME` | Name in the keyring for the account you'll use to broadcast transactions. It defaults to "validator" on local instances of Nibiru. |
| `tx alias` | This "tx" alias will help with reading the tx responses. It defines a command that first extracts the transaction hash using `jq`, waits for 3 seconds to ensure the transaction is processed, queries the transaction details using `nibid q tx`, and appends the structured information to `out.json`. |

```bash
# KEYNAME: Name in the keyring for the account you'll use to broadcast 
# transactions. It defaults to "validator" on local instances of Nibiru.
KEYNAME="validator"    

# This "tx" alias will help with reading the tx responses.
alias tx="jq -rcs '.[0].txhash' | { read txhash; sleep 3; nibid q tx \$txhash | jq '{txhash, height, code, logs, tx, gas_wanted, gas_used}' >> out.json}"
```

### Instantiate the Smart Contract

In order to instantiate a smart contract, you first need a reference its stored
bytecode on the blockchain. The bytecode may already be stored at a known
"code_id". 

If it is not, you must deploy or store the bytecode yourself like so:
```bash

nibid tx wasm store ../../artifacts/nibi_stargate.wasm --from="$KEYNAME" --gas=2000999 -y | tx
```

ℹ️a If smart contract bytecode is already stored, or deployed, it doesn't need to
be redeployed to create new smart contracts. **Instantiating is what creates new
contracts with new state and a unique address.

```bash
CODE_ID=1
nibid tx wasm inst $CODE_ID '{}' --label="fungible tokens" --no-admin --from="$KEYNAME" -y | tx 
```

- Why is the argument `'{}'` passed during instantiation?
    The `InstantiateMsg` for this smart contract is blank.
    ```rust
    #[cw_serde]
    pub struct InstantiateMsg {}    // nibi-stargate/src/msgs.rs
    ```

## Broadcasting Transactions to the Network

```rust
#[cw_serde]
pub enum ExecuteMsg {
    // For x/tokenfactory
    CreateDenom { subdenom: String },
    Mint { coin: Coin, mint_to: String },
    Burn { coin: Coin, burn_from: String },
    ChangeAdmin { denom: String, new_admin: String },
}
```

### Examples

`ExecuteMsg::CreateDenom`

```json
{
  "create_denom": { "subdenom": "zzz" }
}
```

`ExecuteMsg::Mint`

```json
{ 
  "mint": { 
    "coin": { "amount": "[amount]", "denom": "tf/[contract-addr]/[subdenom]" }, 
    "mint_to": "[mint-to-addr]" 
  } 
}
```

`ExecuteMsg::Burn`

```json
{ 
  "burn": { 
    "coin": { "amount": "[amount]", "denom": "tf/[contract-addr]/[subdenom]" }, 
    "burn_from": "[burn-from-addr]" 
  } 
}
```

`ExecuteMsg::ChangeAdmin`

```json
{ 
  "change_admin": { 
    "denom": "tf/[contract-addr]/[subdenom]", 
    "new_admin": "[ADDR]" 
  } 
}
```

## Appendix: Stargate Types

1.  **Stargate messages**: Instances of the [`CosmosMsg::Stargate` variant in
    cosmwasm-std](https://docs.rs/cosmwasm-std/1.4.1/cosmwasm_std/enum.CosmosMsg.html). These correspond to
    [StargateMsg in CosmWasm/wasmvm](https://pkg.go.dev/github.com/CosmWasm/wasmvm@v1.4.1/types#StargateMsg) 
2.  **Stargate queries**: Instances of the [`QueryRequest::Stargate` variant in
    cosmwasm-std](https://docs.rs/cosmwasm-std/1.4.1/cosmwasm_std/enum.QueryRequest.html). These correspond to
    [StargateQuery from CosmWasm/wasmvm](https://pkg.go.dev/github.com/CosmWasm/wasmvm@v1.4.1/types#StargateMsg)

### cosmwasm-std: CosmosMsg::Stargate

```rust
pub enum CosmosMsg<T = Empty> {
    /// ... other variants like Bank, Custom, Staking, Ibc, Wasm
    Stargate {
        type_url: String,
        value: Binary,
    },
}
```

### wasmvm: StargateMsg

```go
type StargateMsg struct {
	TypeURL string `json:"type_url"`
	Value   []byte `json:"value"`
}
```

### cosmwasm-std: QueryRequest::Stargate

```rust
pub enum QueryRequest<C> {
    /// ... other variants like Bank, Custom, Staking, Ibc, Wasm
    Stargate {
        /// this is the fully qualified service path used for routing,
        /// eg. custom/cosmos_sdk.x.bank.v1.Query/QueryBalance
        path: String,
        /// this is the expected protobuf message type (not any), binary encoded
        data: Binary,
    },
}
```

### wasmvm: StargateQuery

```go
// StargateQuery is encoded the same way as abci_query, with path and protobuf
// encoded request data. The format is defined in
// [ADR-21](https://github.com/cosmos/cosmos-sdk/blob/master/docs/architecture/adr-021-protobuf-query-encoding.md).
// The response is protobuf encoded data directly without a JSON response
// wrapper. The caller is responsible for compiling the proper protobuf
// definitions for both requests and responses.
type StargateQuery struct {
	// this is the fully qualified service path used for routing,
	// eg. custom/cosmos_sdk.x.bank.v1.Query/QueryBalance
	Path string `json:"path"`
	// this is the expected protobuf message type (not any), binary encoded
	Data []byte `json:"data"`
}
```