---
order: 2
description: >-
  Nibiru's Wasm module runs CosmWasm smart contracts. Use transaction messages
  (TxMsgs) to deploy contracts, call them, and update module parameters. The
  module emits events that indexers can use to track contract activity.
---

# Module: Wasm

{{ $frontmatter.description }}

## Contents

- [Contents](#contents)
- [Events](#events)
- [Transaction Messages (TxMsgs)](#transaction-messages-txmsgs)

Reference:
- [CosmWasm/wasmd v0.50.0 Golang Docs](https://pkg.go.dev/github.com/CosmWasm/wasmd@v0.50.0)


## Events

Indexers can use Wasm events to track [TxMsgs](../tx-msgs.md) for smart contract
operations.

The SDK emits a `message` event whose `action` attribute is the TxMsg's
Protobuf type URL. To find transactions that submit a `MsgInstantiateContract`,
filter for `message.action='/cosmwasm.wasm.v1.MsgInstantiateContract'`.

The Wasm module emits a separate `instantiate` event for each new contract
instance. Filter by `instantiate.code_id` to find instances of specific code.
Read the `_contract_address` attribute to get the new contract's address.

```json
{
  "type": "instantiate",
  "attributes": [
    {
      "key": "_contract_address",
      "value": "nibi14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr",
      "index": true
    },
    {
      "key": "code_id",
      "value": "1",
      "index": true
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

