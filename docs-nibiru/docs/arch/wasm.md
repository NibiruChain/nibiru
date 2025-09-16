---
order: 2
---

# Module: Wasm  

The `wasm` module of Nibiru Chain allows for executing CosmWasm smart contracts, enabling developers to build decentralized applications (dApps) with transaction messages for contract deployment, execution, and parameter updates. It emits events for transaction indexing, aiding contract interaction tracking and platform adoption.
{synopsis}

## Contents

- [Contents](#contents)
- [Events](#events)
- [Transaction Messages (TxMsgs)](#transaction-messages-txmsgs)

Reference:
- [CosmWasm/wasmd v0.50.0 Golang Docs](https://pkg.go.dev/github.com/CosmWasm/wasmd@v0.50.0)


## Events

A number of events are returned to allow transaction messages (TxMsgs) from smart
contracts to be indexed.

The module for each event is "wasm", and `code_id` is only present when
instantiating a contract so that users can subscribe to new contract instances.
The `code_id` is omitted during invocation (`wasm.Execute`). These events are
emitted any time a contract returns non-empty event attributes.

There is also an "action" field that is added automatically and has a value of either `store-code`, `instantiate` or `execute` depending on which message was sent:

Ex: Instantiate Event

```json
{
    "Type": "message",
    "Attr": [
        {
            "key": "module",
            "value": "wasm"
        },
        {
            "key": "action",
            "value": "instantiate"
        },
        {
            "key": "signer",
            "value": "nibi1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"
        },
        {
            "key": "code_id",
            "value": "1"
        },
        {
            "key": "_contract_address",
            "value": "nibi14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"
        }
    ]
}
```

## Transaction Messages (TxMsgs)

| TxMsg: Wasm                      | Description                                                                                                       |
|------------------------------------|-------------------------------------------------------------------------------------------------------------------|
| `MsgStoreCode`                     | MsgStoreCode to submit Wasm code to the system                                                                       |
| `MsgInstantiateContract`           | MsgInstantiateContract creates a new smart contract instance for the given code id.                                   |
| `MsgInstantiateContract2`          | MsgInstantiateContract2 creates a new smart contract instance for the given code id with a predictable address       |
| `MsgExecuteContract`               | MsgExecute submits the given message data to a smart contract                                                        |
| `MsgMigrateContract`               | MsgMigrate runs a code upgrade/downgrade for a smart contract                                                        |
| `MsgUpdateAdmin`                   | MsgUpdateAdmin sets a new admin for a smart contract                                                                 |
| `MsgClearAdmin`                    | MsgClearAdmin removes any admin stored for a smart contract                                                           |
| `MsgUpdateInstantiateConfig`       | MsgUpdateInstantiateConfig updates instantiate config for a smart contract                                           |
| `MsgUpdateParams`                  | MsgUpdateParams defines a governance operation for updating the x/wasm module parameters                              |
| `MsgSudoContract`                  | MsgSudoContract defines a governance operation for calling sudo on a contract                                        |
| `MsgPinCodes`                      | MsgPinCodes defines a governance operation for pinning a set of code ids in the wasmvm cache                         |
| `MsgUnpinCodes`                    | MsgUnpinCodes defines a governance operation for unpinning a set of code ids in the wasmvm cache                     |
| `MsgStoreAndInstantiateContract`   | MsgStoreAndInstantiateContract defines a governance operation for storing and instantiating the contract             |
| `MsgRemoveCodeUploadParamsAddresses` | MsgRemoveCodeUploadParamsAddresses defines a governance operation for removing addresses from code upload params     |
| `MsgAddCodeUploadParamsAddresses`  | MsgAddCodeUploadParamsAddresses defines a governance operation for adding addresses to code upload params            |
| `MsgStoreAndMigrateContract`       | MsgStoreAndMigrateContract defines a governance operation for storing and migrating the contract                     |
| `MsgUpdateContractLabel`           | MsgUpdateContractLabel sets a new label for a smart contract                                                         |

The full list of messages is described by the module's [gRPC `MsgClient`](https://pkg.go.dev/github.com/CosmWasm/wasmd@v0.50.0/x/wasm/types#MsgClient).

