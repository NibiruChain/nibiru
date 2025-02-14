# devgas

The `devgas` module of Nibiru Chain shares contract execution fees with smart contract
developers. 

This aims to increase the adoption of Nibiru by offering CosmWasm smart
contract developers a direct source of income based on usage.  Developers can
register their smart contracts and every time someone interacts with a
registered smart contract, the contract deployer or their assigned withdrawal
account receives a part of the transaction fees.

Table of Contents

- [Register a Contract Withdrawal Address](#register-a-contract-withdrawal-address)
  - [Register Args](#register-args)
  - [Description](#description)
  - [Permissions](#permissions)
  - [Exceptions](#exceptions)
- [Update a Contract's Withdrawal Address](#update-a-contracts-withdrawal-address)
  - [Update Exception](#update-exception)
- [Concepts](#concepts)
  - [FeeShare](#feeshare)
  - [Registration](#registration)
  - [Fee Distribution](#fee-distribution)
  - [WASM Transaction Fees](#wasm-transaction-fees)
- [State](#state)
  - [State: FeeShare](#state-feeshare)
    - [State: ContractAddress](#state-contractaddress)
    - [DeployerAddress](#deployeraddress)
    - [WithdrawerAddress](#withdraweraddress)
  - [Genesis State](#genesis-state)
- [State Transitions](#state-transitions)
- [Register Fee Share](#register-fee-share)
  - [Update Fee Split](#update-fee-split)
  - [Cancel Fee Split](#cancel-fee-split)
- [TxMsgs - devgas](#txmsgs---devgas)
- [`MsgRegisterFeeShare`](#msgregisterfeeshare)
  - [`MsgUpdateFeeShare`](#msgupdatefeeshare)
  - [`MsgCancelFeeShare`](#msgcancelfeeshare)
- [Ante](#ante)
- [Handling](#handling)
- [Events](#events)
  - [Event: Register Fee Split](#event-register-fee-split)
  - [Event: Update Fee Split](#event-update-fee-split)
  - [Event: Cancel Fee Split](#event-cancel-fee-split)
- [Module Parameters](#module-parameters)
- [Enable FeeShare Module](#enable-feeshare-module)
  - [Developer Shares Amount](#developer-shares-amount)
  - [Allowed Denominations](#allowed-denominations)
- [Clients](#clients)
- [Command Line Interface](#command-line-interface)
  - [Queries](#queries)
  - [Transactions](#transactions)
- [gRPC Queries](#grpc-queries)
  - [gRPC Transactions](#grpc-transactions)

# Register a Contract Withdrawal Address

```bash
nibid tx devgas register [contract_bech32] [withdraw_bech32] --from [key]
```

Registers the withdrawal address for the given contract.

### Register Args

`contract_bech32 (string, required)`: The bech32 address of the contract whose
interaction fees will be shared.

`withdraw_bech32 (string, required)`: The bech32 address where the interaction
fees will be sent every block.


### Description

This command registers the withdrawal address for the given contract. Any time
a user interacts with your contract, the funds will be sent to the withdrawal
address. It can be any valid address, such as a DAO, normal account, another
contract, or a multi-sig.

### Permissions

This command can only be run by the admin of the contract. If there is no
admin, then it can only be run by the contract creator.

### Exceptions

- `withdraw_bech32` can not be the community pool (distribution) address. This
  is a limitation of the way the SDK handles this module account

- For contracts created or administered by a contract factory, the withdrawal
  address can only be the same as the contract address. This can be registered
  by anyone, but it's unchangeable. This is helpful for SubDAOs or public goods
  to save fees in the treasury.

If you create a contract like this, it's best to create an execution method for
withdrawing fees to an account. To do this, you'll need to save the withdrawal
address in the contract's state before uploading a non-migratable contract.

## Update a Contract's Withdrawal Address

This can be changed at any time so long as you are still the admin or creator
of a contract with the command:

```bash
nibid tx devgas update [contract] [new_withdraw_address]
```

### Update Exception

This can not be done if the contract was created from or is administered by
another contract (a contract factory). There is not currently a way for a
contract to change its own withdrawal address directly.

# Concepts

### FeeShare

The DevGas (`x/devgas`) module is a revenue-per-gas model, which allows
developers to get paid for deploying their decentralized applications (dApps)
on Nibiru. This helps developers to generate revenue every time a user
invokes their contracts to execute a transaction on the chain. 

This registration is permissionless to sign up for and begin earning fees from.
By default, 50% of all gas fees for Execute Messages are shared. This
can be changed by governance and implemented by the `x/devgas` module.

### Registration

Developers register their contract applications to gain their cut of fees per
execution. Any contract can be registered by a developer by submitting a signed
transaction. After the transaction is executed successfully, the developer will
start receiving a portion of the transaction fees paid when a user interacts
with the registered contract. The developer can have the funds sent to their
wallet, a DAO, or any other wallet address on the Nibiru.

::: tip
**NOTE**: If your contract is part of a development project, please ensure that
the deployer of the contract (or the factory/DAO that deployed the contract) is
an account that is owned by that project. This avoids the situation, that an
individual deployer who leaves your project could become malicious.
:::

### Fee Distribution

As described above, developers will earn a portion of the transaction fee after
registering their contracts. To understand how transaction fees are
distributed, we will look at the following in detail:

* The transactions eligible are only [Wasm Execute Txs](https://github.com/CosmWasm/wasmd/blob/main/proto/cosmwasm/wasm/v1/tx.proto#L115-L127) (`MsgExecuteContract`).

### WASM Transaction Fees

Users pay transaction fees to pay to interact with smart contracts on Nibiru.
When a transaction is executed, the entire fee amount (`gas limit * gas price`)
is sent to the `FeeCollector` module account during the [Cosmos SDK
AnteHandler](https://docs.cosmos.network/main/modules/auth/#antehandlers)
execution. 

After this step, the `FeeCollector` sends 50% of the funds and splits them
between contracts that were executed on the transaction. If the fees paid are
not accepted by governance, there is no payout to the developers (for example,
niche base tokens) for tax purposes. If a user sends a message and it does not
interact with any contracts (ex: bankSend), then the entire fee is sent to the
`FeeCollector` as expected.

# State

The `x/devgas` module keeps the following objects in the state:

| State Object          | Description                           | Key                                                               | Value              | Store |
| :-------------------- | :------------------------------------ | :---------------------------------------------------------------- | :----------------- | :---- |
| `FeeShare`            | Fee split bytecode                    | `[]byte{1} + []byte(contract_address)`                            | `[]byte{feeshare}` | KV    |
| `DeployerFeeShares`   | Contract by deployer address bytecode | `[]byte{2} + []byte(deployer_address) + []byte(contract_address)` | `[]byte{1}`        | KV    |
| `FeeSharesByWithdrawer` | Contract by withdraw address bytecode | `[]byte{3} + []byte(withdraw_address) + []byte(contract_address)` | `[]byte{1}`        | KV    |

### State: FeeShare

A `FeeShare` defines an instance that organizes fee distribution conditions for
the owner of a given smart contract

```go
type FeeShare struct {
  // contract_address is the bech32 address of a registered contract in string form
  ContractAddress string `protobuf:"bytes,1,opt,name=contract_address,json=contractAddress,proto3" json:"contract_address,omitempty"`
  // deployer_address is the bech32 address of message sender. It must be the
  // same as the contracts admin address.
  DeployerAddress string `protobuf:"bytes,2,opt,name=deployer_address,json=deployerAddress,proto3" json:"deployer_address,omitempty"`
  // withdrawer_address is the bech32 address of account receiving the
  // transaction fees.
  WithdrawerAddress string `protobuf:"bytes,3,opt,name=withdrawer_address,json=withdrawerAddress,proto3" json:"withdrawer_address,omitempty"`
}
```

#### State: ContractAddress

`ContractAddress` defines the contract address that has been registered for fee distribution.

#### DeployerAddress

A `DeployerAddress` is the admin address for a registered contract.

#### WithdrawerAddress

The `WithdrawerAddress` is the address that receives transaction fees for a registered contract.

### Genesis State

The `x/devgas` module's `GenesisState` defines the state necessary for initializing the chain from a previously exported height. It contains the module parameters and the fee share for registered contracts:

```go
// GenesisState defines the module's genesis state.
type GenesisState struct {
  // module parameters
  Params Params `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
  // active registered contracts for fee distribution
  FeeShares []FeeShare `protobuf:"bytes,2,rep,name=feeshares,json=feeshares,proto3" json:"feeshares"`
}
```

# State Transitions

The `x/devgas` module allows for three types of state transitions:
`RegisterFeeShare`, `UpdateFeeShare` and `CancelFeeShare`. The logic for
distributing transaction fees is handled through the [Ante
handler](/app/ante.go).

## Register Fee Share

A developer registers a contract for receiving transaction fees by defining the
contract address and the withdrawal address for fees to be paid too. If this is
not set, the developer can not get income from the contract. This is opt-in for
tax purposes. When registering for fees to be paid, you MUST be the admin of
said wasm contract. The withdrawal address can be the same as the contract's
address if you so choose.

1. User submits a `RegisterFeeShare` to register a contract address, along with
   a withdrawal address that they would like to receive the fees to
2. Check if the following conditions pass:
    1. `x/devgas` module is enabled via Governance
    2. the contract was not previously registered
    3. deployer has a valid account (it has done at least one transaction)
    4. the contract address exists
    5. the deployer signing the transaction is the admin of the contract
    6. the contract is already deployed
3. Store an instance of the provided share.

All transactions sent to the registered contract occurring after registration
will have their fees distributed to the developer, according to the global
`DeveloperShares` parameter in governance.

### Update Fee Split

A developer updates the withdraw address for a registered contract, defining
the contract address and the new withdraw address.

1. The user submits a `UpdateFeeShare`
2. Check if the following conditions pass:
    1. `x/devgas` module is enabled
    2. the contract is registered
    3. the signer of the transaction is the same as the contract admin per the
       WasmVM
3. Update the fee with the new withdrawal address.

After this update, the developer receives the fees on the new withdrawal
address.

### Cancel Fee Split

A developer cancels receiving fees for a registered contract, defining the
contract address.

1. The user submits a `CancelFeeShare`
2. Check if the following conditions pass:
    1. `x/devgas` module is enabled
    2. the contract is registered
    3. the signer of the transaction is the same as the contract admin per the
       WasmVM
3. Remove share from storage

The developer no longer receives fees from transactions sent to this contract.
All fees go to the community.

# TxMsgs - devgas

This section defines the `sdk.Msg` concrete types that result in the state
transitions defined on the previous section.

## `MsgRegisterFeeShare`

Defines a transaction signed by a developer to register a contract for
transaction fee distribution. The sender must be an EOA that corresponds to the
contract deployer address.

```go
type MsgRegisterFeeShare struct {
  // contract_address in bech32 format
  ContractAddress string `protobuf:"bytes,1,opt,name=contract_address,json=contractAddress,proto3" json:"contract_address,omitempty"`
  // deployer_address is the bech32 address of message sender. It must be the
  // same the contract's admin address
  DeployerAddress string `protobuf:"bytes,2,opt,name=deployer_address,json=deployerAddress,proto3" json:"deployer_address,omitempty"`
  // withdrawer_address is the bech32 address of account receiving the
  // transaction fees
  WithdrawerAddress string `protobuf:"bytes,3,opt,name=withdrawer_address,json=withdrawerAddress,proto3" json:"withdrawer_address,omitempty"`
}
```

The message content stateless validation fails if:

- Contract bech32 address is invalid
- Deployer bech32 address is invalid
- Withdraw bech32 address is invalid

### `MsgUpdateFeeShare`

Defines a transaction signed by a developer to update the withdraw address of a contract registered for transaction fee distribution. The sender must be the admin of the contract.

```go
type MsgUpdateFeeShare struct {
  // contract_address in bech32 format
  ContractAddress string `protobuf:"bytes,1,opt,name=contract_address,json=contractAddress,proto3" json:"contract_address,omitempty"`
  // deployer_address is the bech32 address of message sender. It must be the
  // same the contract's admin address
  DeployerAddress string `protobuf:"bytes,2,opt,name=deployer_address,json=deployerAddress,proto3" json:"deployer_address,omitempty"`
  // withdrawer_address is the bech32 address of account receiving the
  // transaction fees
  WithdrawerAddress string `protobuf:"bytes,3,opt,name=withdrawer_address,json=withdrawerAddress,proto3" json:"withdrawer_address,omitempty"`
}
```

The message content stateless validation fails if:

- Contract bech32 address is invalid
- Deployer bech32 address is invalid
- Withdraw bech32 address is invalid

### `MsgCancelFeeShare`

Defines a transaction signed by a developer to remove the information for a registered contract. Transaction fees will no longer be distributed to the developer for this smart contract. The sender must be an admin that corresponds to the contract.

```go
type MsgCancelFeeShare struct {
  // contract_address in bech32 format
  ContractAddress string `protobuf:"bytes,1,opt,name=contract_address,json=contractAddress,proto3" json:"contract_address,omitempty"`
  // deployer_address is the bech32 address of message sender. It must be the
  // same the contract's admin address
  DeployerAddress string `protobuf:"bytes,2,opt,name=deployer_address,json=deployerAddress,proto3" json:"deployer_address,omitempty"`
}
```

The message content stateless validation fails if:

- Contract bech32 address is invalid
- Contract bech32 address is zero
- Deployer bech32 address is invalid

# Ante

The fees module uses the ante handler to distribute fees between developers and the community.

## Handling

An [Ante Decorator](/x/devgas/ante/ante.go) executes custom logic after each
successful WasmExecuteMsg transaction. All fees paid by a user for transaction
execution are sent to the `FeeCollector` module account during the
`AnteHandler` execution before being redistributed to the registered contract
developers.

If the `x/devgas` module is disabled or the Wasm Execute Msg transaction
targets an unregistered contract, the handler returns `nil`, without performing
any actions. In this case, 100% of the transaction fees remain in the
`FeeCollector` module, to be distributed elsewhere.

If the `x/devgas` module is enabled and a Wasm Execute Msg transaction
targets a registered contract, the handler sends a percentage of the
transaction fees (paid by the user) to the withdraw address set for that
contract.

1. The user submits an Execute transaction (`MsgExecuteContract`) to a smart
   contract and the transaction is executed successfully
2. Check if
   * fees module is enabled
   * the smart contract is registered to receive fee split
3. Calculate developer fees according to the `DeveloperShares` parameter.
4. Check what fees governance allows to be paid in
5. Check which contracts the user executed that also have been registered.
6. Calculate the total amount of fees to be paid to the developer(s). If
multiple, split the 50% between all registered withdrawal addresses.
7. Distribute the remaining amount in the `FeeCollector` to validators
according to the [SDK  Distribution
Scheme](https://docs.cosmos.network/main/modules/distribution/03_begin_block.html#the-distribution-scheme).

# Events

The `x/devgas` module emits the following events:

### Event: Register Fee Split

| Type                 | Attribute Key          | Attribute Value           |
| :------------------- | :--------------------- | :------------------------ |
| `register_feeshare`  | `"contract"`            | `{msg.ContractAddress}`   |
| `register_feeshare`  | `"sender"`              | `{msg.DeployerAddress}`   |
| `register_feeshare`  | `"withdrawer_address"`  | `{msg.WithdrawerAddress}` |

### Event: Update Fee Split

| Type               | Attribute Key          | Attribute Value           |
| :----------------- | :--------------------- | :------------------------ |
| `update_feeshare`  | `"contract"`            | `{msg.ContractAddress}`   |
| `update_feeshare`  | `"sender"`              | `{msg.DeployerAddress}`   |
| `update_feeshare`  | `"withdrawer_address"`  | `{msg.WithdrawerAddress}` |

### Event: Cancel Fee Split

| Type               | Attribute Key | Attribute Value         |
| :----------------- | :------------ | :---------------------- |
| `cancel_feeshare`  | `"contract"`   | `{msg.ContractAddress}` |
| `cancel_feeshare`  | `"sender"`     | `{msg.DeployerAddress}` |

# Module Parameters

The fee Split module contains the following parameters:

| Key                        | Type        | Default Value    |
| :------------------------- | :---------- | :--------------- |
| `EnableFeeShare`           | bool        | `true`           |
| `DeveloperShares`          | sdk.Dec     | `50%`            |
| `AllowedDenoms`            | []string{}  | `[]string(nil)`  |

## Enable FeeShare Module

The `EnableFeeShare` parameter toggles all state transitions in the module.
When the parameter is disabled, it will prevent any transaction fees from being
distributed to contract deplorers and it will disallow contract registrations,
updates or cancellations.

### Developer Shares Amount

The `DeveloperShares` parameter is the percentage of transaction fees that are
sent to the contract deplorers.

### Allowed Denominations

The `AllowedDenoms` parameter is used to specify which fees coins will be paid
to contract developers. If this is empty, all fees paid will be split. If not,
only fees specified here will be paid out to the withdrawal address.

# Clients

## Command Line Interface

Find below a list of `nibid` commands added with the  `x/devgas` module. You
can obtain the full list by using the `nibid -h` command. A CLI command can
look like this:

```bash
nibid query feeshare params
```

### Queries

| Command            | Subcommand             | Description                              |
| :----------------- | :--------------------- | :--------------------------------------- |
| `query` `feeshare` | `params`               | Get devgas params                      |
| `query` `feeshare` | `contract`             | Get the devgas for a given contract    |
| `query` `feeshare` | `contracts`            | Get all feeshares                        |
| `query` `feeshare` | `deployer-contracts`   | Get all feeshares of a given deployer    |
| `query` `feeshare` | `withdrawer-contracts` | Get all feeshares of a given withdrawer  |

### Transactions

| Command         | Subcommand | Description                                |
| :-------------- | :--------- | :----------------------------------------- |
| `tx` `feeshare` | `register` | Register a contract for receiving devgas |
| `tx` `feeshare` | `update`   | Update the withdraw address for a contract |
| `tx` `feeshare` | `cancel`   | Remove the devgas for a contract         |

## gRPC Queries

| Verb   | Method                                            | Description                              |
| :----- | :------------------------------------------------ | :--------------------------------------- |
| `gRPC` | `nibiru.devgas.v1.Query/Params`                   | Get devgas params                      |
| `gRPC` | `nibiru.devgas.v1.Query/FeeShare`                  | Get the devgas for a given contract    |
| `gRPC` | `nibiru.devgas.v1.Query/FeeShares`                 | Get all feeshares                        |
| `gRPC` | `nibiru.devgas.v1.Query/DeployerFeeShares`         | Get all feeshares of a given deployer    |
| `gRPC` | `nibiru.devgas.v1.Query/FeeSharesByWithdrawer`       | Get all feeshares of a given withdrawer  |
| `GET`  | `/nibiru.devgas/v1/params`                        | Get devgas params                      |
| `GET`  | `/nibiru.devgas/v1/feeshares/{contract_address}`  | Get the devgas for a given contract    |
| `GET`  | `/nibiru.devgas/v1/feeshares`                     | Get all feeshares                        |
| `GET`  | `/nibiru.devgas/v1/feeshares/{deployer_address}`  | Get all feeshares of a given deployer    |
| `GET`  | `/nibiru.devgas/v1/feeshares/{withdraw_address}`  | Get all feeshares of a given withdrawer  |

### gRPC Transactions

| Verb   | Method                                     | Description                                |
| :----- | :----------------------------------------- | :----------------------------------------- |
| `gRPC` | `nibiru.devgas.v1.Msg/RegisterFeeShare`   | Register a contract for receiving devgas   |
| `gRPC` | `nibiru.devgas.v1.Msg/UpdateFeeShare`     | Update the withdraw address for a contract   |
| `gRPC` | `nibiru.devgas.v1.Msg/CancelFeeShare`     | Remove the devgas for a contract           |
| `POST` | `/nibiru.devgas/v1/tx/register_feeshare` | Register a contract for receiving devgas   |
| `POST` | `/nibiru.devgas/v1/tx/update_feeshare`   | Update the withdraw address for a contract   |
| `POST` | `/nibiru.devgas/v1/tx/cancel_feeshare`   | Remove the devgas for a contract           |

## Credits: Evmos and Juno

> "This module is a heavily modified fork of
[evmos/x/revenue](https://github.com/evmos/evmos/tree/main/x/revenue)" - Juno Network

This module is a heavily modified fork of Juno's heavily modified fork. 🙃
