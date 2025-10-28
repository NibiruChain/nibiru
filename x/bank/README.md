# Nibiru Bank Module

## Abstract

The **Bank** module is responsible for managing all token balances and transfers
on Nibiru.  It handles multi-asset accounting, module-controlled minting and
burning, and enforces total supply invariants across the chain.

Nibiru extends the Cosmos SDK `x/bank` with high-precision accounting for NIBI,
the chain's native token.  Balances are represented through a **dual-balance
model** combining standard bank units (`unibi`, 10⁶) and wei sub-units (10¹²) for
full 18-decimal precision.  This enables accurate integration with Nibiru's EVM
environment while maintaining Cosmos SDK compatibility.

The module provides:
1. Multi-asset bank coin transfers:   The bank module is responsible for handling
multi-asset coin transfers between accounts and tracking special-case
pseudo-transfers which must work differently with particular kinds of accounts
(notably delegating/undelegating for vesting accounts). It exposes several
interfaces with varying capabilities for secure interaction with other modules
which must alter user balances.
1. Secure mechanisms for coin transfers, minting, and burning.
1. Support for module accounts with scoped permissions.
1. Accurate total supply tracking with relaxed invariants for NIBI.
1. gRPC and CLI interfaces for querying balances, supply, and parameters.

The `x/bank` module forms the foundation of asset management in Nibiru's multi-VM
architecture, ensuring all on-chain tokens—including NIBI, staked derivatives,
and module-held reserves—are handled consistently and securely.

#### Contents

- [Abstract](#abstract)
- [Contents](#contents)
- [Supply](#supply)
  - [Total Supply](#total-supply)
- [Module Accounts](#module-accounts)
- [State](#state)
- [Nibiru Extensions to `x/bank`](#nibiru-extensions-to-xbank)
- [Module Params](#module-params)
- [Security Considerations](#security-considerations)
- [Keepers](#keepers)
  - [Common Types](#common-types)
    - [Input](#input)
    - [Output](#output)
- [Messages](#messages)
- [Events](#events)
- [Parameters](#parameters)
- [Command Line Interface (CLI)](#command-line-interface-cli)
  - [CLI Queries](#cli-queries)
  - [CLI Transactions](#cli-transactions)
- [gRPC](#grpc)

## Supply

The `supply` functionality:

* passively tracks the total supply of coins within a chain,
* provides a pattern for modules to hold/interact with `Coins`, and
* introduces the invariant check to verify a chain's total supply.

### Total Supply

The total `Supply` of the network is equal to the sum of all coins from the
account. The total supply is updated every time a `Coin` is minted (eg: as part
of the inflation mechanism) or burned (eg: due to slashing or if a governance
proposal is vetoed).

## Module Accounts

The supply functionality introduces a new type of `auth.Account` which can be used by
modules to allocate tokens and in special cases mint or burn tokens. At a base
level these module accounts are capable of sending/receiving tokens to and from
`auth.Account`s and other module accounts. This design replaces previous
alternative designs where, to hold tokens, modules would burn the incoming
tokens from the sender account, and then track those tokens internally. Later,
in order to send tokens, the module would need to effectively mint tokens
within a destination account. The new design removes duplicate logic between
modules to perform this accounting.

The `ModuleAccount` interface is defined as follows:

```go
type ModuleAccount interface {
  auth.Account               // same methods as the Account interface

  GetName() string           // name of the module; used to obtain the address
  GetPermissions() []string  // permissions of module account
  HasPermission(string) bool
}
```

The supply `Keeper` also introduces new wrapper functions for the auth `Keeper`
and the bank `Keeper` that are related to `ModuleAccount`s in order to be able
to:

* Get and set `ModuleAccount`s by providing the `Name`.
* Send coins from and to other `ModuleAccount`s or standard `Account`s
  (`BaseAccount` or `VestingAccount`) by passing only the `Name`.
* `Mint` or `Burn` coins for a `ModuleAccount` (restricted to its permissions).

## State

The `x/bank` module keeps state of the following primary objects:

1. Account balances
2. Denomination metadata
3. The total supply of all balances
4. Information on which denominations are allowed to be sent.

In addition, the `x/bank` module keeps the following indexes to manage the
aforementioned state:

* Supply Index: `0x0 | byte(denom) -> byte(amount)`
* Denom Metadata Index: `0x1 | byte(denom) -> ProtocolBuffer(Metadata)`
* Balances Index: `0x2 | byte(address length) | []byte(address) | []byte(balance.Denom) -> ProtocolBuffer(balance)`
* Reverse Denomination to Address Index: `0x03 | byte(denom) | 0x00 | []byte(address) -> 0`

Here's a tightened, Google-style rewrite that keeps the key sections you liked and trims the rest for focus and clarity:

---

## Nibiru Extensions to `x/bank`

### What's different on Nibiru

Nibiru extends the Cosmos SDK `x/bank` with a **dual-balance model** for NIBI and precise wei-level accounting.

* **Dual balance:**
  Balances are split into `unibi` (6 decimals) and a wei remainder store (12 decimals). Together they form a unified 18-decimal balance.
* **Wei APIs:**
  `AddWei`, `SubWei`, and `GetWeiBalance` operate in wei. Normalization handles carry and borrow at the 10¹² boundary.
* **Per-block delta:**
  `WeiBlockDelta` records the net wei added or subtracted each block for EVM supply reconciliation.
* **Invariant relaxation:**
  The total supply invariant ignores NIBI mismatches, allowing minor wei-store differences.
* **Events:**
  Any wei-affecting action emits `EventWeiChange` with a reason code.

### Units and notation

| Unit          | Description                    | Scale         |
| ------------- | ------------------------------ | ------------- |
| `unibi`       | base x/bank denomination       | 10⁶ = 1 NIBI  |
| `wei`         | smallest sub-unit for EVM math | 10¹⁸ = 1 NIBI |
| `WeiPerUnibi` | conversion factor              | 10¹²          |

For account A:
`agg_wei(A) = unibi_balance(A) * 10¹² + wei_store(A)`
where `wei_store(A)` ∈ [0, 10¹²).

### Storage layout

| Namespace                            | Key                | Purpose                     |
| ------------------------------------ | ------------------ | --------------------------- |
| `NAMESPACE_BALANCE_WEI (15)`         | address → Int      | Per-account wei remainder   |
| `NAMESPACE_WEI_BLOCK_DELTA (16)`     | transient          | Net Δwei this block         |
| `NAMESPACE_WEI_COMMITTED_DELTA (17)` | persistent         | Historical deltas for audit |
| `StoreKeyTransient`                  | `"transient_bank"` | Transient store key         |

### Public API

```go
type NibiruExtKeeper interface {
  AddWei(ctx sdk.Context, addr sdk.AccAddress, amtWei *uint256.Int)
  GetWeiBalance(ctx sdk.Context, addr sdk.AccAddress) *uint256.Int
  SubWei(ctx sdk.Context, addr sdk.AccAddress, amtWei *uint256.Int) error
  WeiBlockDelta(ctx sdk.Context) sdkmath.Int
  SumWeiStoreBals(ctx sdk.Context) sdkmath.Int
}
```

### Behavior

* `AddWei` and `SubWei` normalize at 10¹² and no-op on zero.
* `SubWei` errors if the total wei balance is insufficient.
* Every successful call updates `WeiBlockDelta` and emits `EventWeiChange`.

### Example flows

**Carry:**
Start `unibi = 0`, `wei = 0`. `AddWei(1e12 + 420)` → `unibi = 1`, `wei = 420`.

**Borrow:**
Start `unibi = 2`, `wei = 500`. `SubWei(1e12 + 200)` → `unibi = 1`, `wei = 300`.

**Drain:**
`SubWei(total balance)` → `unibi = 0`, `wei = 0`.

### Supply invariant

* Checks equality for all **non-NIBI** coins.
* Reports NIBI totals in both `unibi` and wei, but does not fail on mismatch.
* Used with `SumWeiStoreBals` and `WeiBlockDelta` to confirm total-supply consistency.

### Integration notes

* EVM modules use the wei APIs; other modules can continue with standard `SendCoins`.
* Listeners can track `EventWeiChange` to compute full 18-decimal balances.
* Heavy operations (`SumWeiStoreBals`) should be limited to invariants and crisis checks.

---

## Module Params

The bank module stores it's params in state with the prefix of `0x05`,
it can be updated with governance or the address with authority.

* Params: `0x05 | ProtocolBuffer(Params)`

```protobuf reference
https://github.com/cosmos/cosmos-sdk/blob/v0.47.0-rc1/proto/cosmos/bank/v1beta1/bank.proto#L12-L23
```

## Security Considerations

The bank module holds direct custody over user and module account balances.  
Errors or misconfigurations can cause permanent loss of funds or halted networks.  
Nibiru's implementation follows the Cosmos SDK model but adds additional safeguards and precision accounting for NIBI.

### 1. Module Account Restrictions

- Module accounts have special privileges such as minting or burning coins.  
- **Never allow direct sends to a module account** unless explicitly permitted in its permissions.  
- Use `SendCoinsFromModuleToAccount` and `SendCoinsFromModuleToModule` for any interaction involving module balances.

> End users and client applications should never target module account addresses directly.

### 2. Minting and Burning Controls

- Only modules with the `Minter` or `Burner` permission can modify supply.  
- Minting is performed through `MintCoins(ctx, module, amt)`; burning through `BurnCoins(ctx, module, amt)`.  
- These actions must be deterministic and auditable, as they affect global invariants.  
- The Nibiru invariant excludes NIBI mismatches caused by wei rounding, but all other assets must remain strictly conserved.

### 3. Invariants and Supply Checks

- The `TotalSupply` invariant ensures total balances equal declared supply for all coins other than NIBI.  
- NIBI is treated specially due to its dual-balance model (`unibi + wei_store`).  
- Use invariants to catch silent corruption or bypassed mint/burn paths early.  
- Heavy supply audits using `SumWeiStoreBals` should be limited to the crisis module.

### 4. Governance and Authority

- Parameter updates (`MsgUpdateParams`, `MsgSetSendEnabled`) can only be signed by the `x/gov` module account.  
- Governance proposals changing send permissions, mint limits, or default parameters should be reviewed for economic safety.

### 5. Event Integrity

- Every wei-affecting operation (`AddWei`, `SubWei`, or `SendCoins` on NIBI) emits `EventWeiChange`.  
- Indexers and accounting systems should rely on these events rather than raw bank state for reconciliation.  
- Missing or malformed events can lead to accounting drift between EVM and SDK modules.

### 6. Operational Recommendations

- **Avoid custom mint/burn logic** in downstream modules—use `x/bank` keepers to preserve invariants.  
- **Audit all send paths** to ensure module accounts cannot receive tokens unintentionally.  
- **Enable crisis invariants** in production configurations to catch unexpected supply deltas.

## Keepers

The bank module provides these exported keeper interfaces that can be
passed to other modules that read or update account balances. Modules
should use the least-permissive interface that provides the functionality they
require.

Best practices dictate careful review of `bank` module code to ensure that
permissions are limited in the way that you expect.

### Common Types

#### Input

An input of a multiparty transfer

```protobuf
// Input models transaction input.
message Input {
  string   address                        = 1;
  repeated cosmos.base.v1beta1.Coin coins = 2;
}
```

#### Output

An output of a multiparty transfer.

```protobuf
// Output models transaction outputs.
message Output {
  string   address                        = 1;
  repeated cosmos.base.v1beta1.Coin coins = 2;
}
```

## Messages

### MsgSend

Send coins from one address to another.

```protobuf reference
https://github.com/cosmos/cosmos-sdk/blob/v0.47.0-rc1/proto/cosmos/bank/v1beta1/tx.proto#L38-L53
```

The message will fail under the following conditions:

* The coins do not have sending enabled
* The `to` address is restricted

### MsgMultiSend

Send coins from one sender and to a series of different address. If any of the receiving addresses do not correspond to an existing account, a new account is created.

```protobuf reference
https://github.com/cosmos/cosmos-sdk/blob/v0.47.0-rc1/proto/cosmos/bank/v1beta1/tx.proto#L58-L69
```

The message will fail under the following conditions:

* Any of the coins do not have sending enabled
* Any of the `to` addresses are restricted
* Any of the coins are locked
* The inputs and outputs do not correctly correspond to one another

### MsgUpdateParams

The `bank` module params can be updated through `MsgUpdateParams`, which can be done using governance proposal. The signer will always be the `gov` module account address. 

```protobuf reference
https://github.com/cosmos/cosmos-sdk/blob/v0.47.0-rc1/proto/cosmos/bank/v1beta1/tx.proto#L74-L88
```

The message handling can fail if:

* signer is not the gov module account address.

### MsgSetSendEnabled

Used with the x/gov module to set create/edit SendEnabled entries.

```protobuf reference
https://github.com/cosmos/cosmos-sdk/blob/v0.47.0-rc1/proto/cosmos/bank/v1beta1/tx.proto#L96-L117
```

The message will fail under the following conditions:

* The authority is not a bech32 address.
* The authority is not x/gov module's address.
* There are multiple SendEnabled entries with the same Denom.
* One or more SendEnabled entries has an invalid Denom.

## Events

The bank module emits the following events:

### Message Events

#### MsgSend

| Type     | Attribute Key | Attribute Value    |
| -------- | ------------- | ------------------ |
| transfer | recipient     | {recipientAddress} |
| transfer | amount        | {amount}           |
| message  | module        | bank               |
| message  | action        | send               |
| message  | sender        | {senderAddress}    |

#### MsgMultiSend

| Type     | Attribute Key | Attribute Value    |
| -------- | ------------- | ------------------ |
| transfer | recipient     | {recipientAddress} |
| transfer | amount        | {amount}           |
| message  | module        | bank               |
| message  | action        | multisend          |
| message  | sender        | {senderAddress}    |

### Keeper Events

In addition to message events, the bank keeper will produce events when the following methods are called (or any method which ends up calling them)

#### MintCoins

```json
{
  "type": "coinbase",
  "attributes": [
    {
      "key": "minter",
      "value": "{{sdk.AccAddress of the module minting coins}}",
      "index": true
    },
    {
      "key": "amount",
      "value": "{{sdk.Coins being minted}}",
      "index": true
    }
  ]
}
```

```json
{
  "type": "coin_received",
  "attributes": [
    {
      "key": "receiver",
      "value": "{{sdk.AccAddress of the module minting coins}}",
      "index": true
    },
    {
      "key": "amount",
      "value": "{{sdk.Coins being received}}",
      "index": true
    }
  ]
}
```

#### BurnCoins

```json
{
  "type": "burn",
  "attributes": [
    {
      "key": "burner",
      "value": "{{sdk.AccAddress of the module burning coins}}",
      "index": true
    },
    {
      "key": "amount",
      "value": "{{sdk.Coins being burned}}",
      "index": true
    }
  ]
}
```

```json
{
  "type": "coin_spent",
  "attributes": [
    {
      "key": "spender",
      "value": "{{sdk.AccAddress of the module burning coins}}",
      "index": true
    },
    {
      "key": "amount",
      "value": "{{sdk.Coins being burned}}",
      "index": true
    }
  ]
}
```

#### addCoins

```json
{
  "type": "coin_received",
  "attributes": [
    {
      "key": "receiver",
      "value": "{{sdk.AccAddress of the address beneficiary of the coins}}",
      "index": true
    },
    {
      "key": "amount",
      "value": "{{sdk.Coins being received}}",
      "index": true
    }
  ]
}
```

#### subUnlockedCoins/DelegateCoins

```json
{
  "type": "coin_spent",
  "attributes": [
    {
      "key": "spender",
      "value": "{{sdk.AccAddress of the address which is spending coins}}",
      "index": true
    },
    {
      "key": "amount",
      "value": "{{sdk.Coins being spent}}",
      "index": true
    }
  ]
}
```

## Parameters

The bank module contains the following parameters

### SendEnabled

The SendEnabled parameter is now deprecated and not to be use. It is replaced
with state store records.


### DefaultSendEnabled

The default send enabled value controls send transfer capability for all
coin denominations unless specifically included in the array of `SendEnabled`
parameters.

## Command Line Interface (CLI)


A user can query and interact with the `bank` module using the CLI.

### CLI Transactions

The `tx` commands allow users to interact with the `bank` module.

```bash
nibid tx bank --help
```

##### send

The `send` command allows users to send funds from one account to another.

```bash
nibid tx bank send [from_key_or_address] [to_address] [amount] [flags]
```

Example:

```bash
nibid tx bank send cosmos1.. cosmos1.. 100stake
```


### CLI Queries

The `query` commands allow users to query `bank` state.

```bash
nibid query bank --help
```

##### balances

The `balances` command allows users to query account balances by address.

```bash
nibid query bank balances [address] [flags]
```

Example:

```bash
nibid query bank balances cosmos1..
```

Example Output:

```yml
balances:
- amount: "1000000000"
  denom: stake
pagination:
  next_key: null
  total: "0"
```

##### denom-metadata

The `denom-metadata` command allows users to query metadata for coin denominations. A user can query metadata for a single denomination using the `--denom` flag or all denominations without it.

```bash
nibid query bank denom-metadata [flags]
```

Example:

```bash
nibid query bank denom-metadata --denom stake
```

Example Output:

```yml
metadata:
  base: stake
  denom_units:
  - aliases:
    - STAKE
    denom: stake
  description: native staking token of simulation app
  display: stake
  name: SimApp Token
  symbol: STK
```

##### total

The `total` command allows users to query the total supply of coins. A user can query the total supply for a single coin using the `--denom` flag or all coins without it.

```bash
nibid query bank total [flags]
```

Example:

```bash
nibid query bank total --denom stake
```

Example Output:

```yml
amount: "10000000000"
denom: stake
```

##### send-enabled

The `send-enabled` command allows users to query for all or some SendEnabled entries.

```bash
nibid query bank send-enabled [denom1 ...] [flags]
```

Example:

```bash
nibid query bank send-enabled
```

Example output:

```yml
send_enabled:
- denom: foocoin
  enabled: true
- denom: barcoin
pagination:
  next-key: null
  total: 2 
```

## gRPC

A user can query the `bank` module using gRPC endpoints.

### Query Balance

The `Balance` endpoint allows users to query account balance by address for a given denomination.

```bash
cosmos.bank.v1beta1.Query/Balance
```

Example:

```bash
grpcurl -plaintext \
    -d '{"address":"cosmos1..","denom":"stake"}' \
    localhost:9090 \
    cosmos.bank.v1beta1.Query/Balance
```

Example Output:

```json
{
  "balance": {
    "denom": "stake",
    "amount": "1000000000"
  }
}
```

### Query AllBalances

The `AllBalances` endpoint allows users to query account balance by address for all denominations.

```bash
cosmos.bank.v1beta1.Query/AllBalances
```

Example:

```bash
grpcurl -plaintext \
    -d '{"address":"cosmos1.."}' \
    localhost:9090 \
    cosmos.bank.v1beta1.Query/AllBalances
```

Example Output:

```json
{
  "balances": [
    {
      "denom": "stake",
      "amount": "1000000000"
    }
  ],
  "pagination": {
    "total": "1"
  }
}
```

### Query DenomMetadata

The `DenomMetadata` endpoint allows users to query metadata for a single coin denomination.

```bash
cosmos.bank.v1beta1.Query/DenomMetadata
```

Example:

```bash
grpcurl -plaintext \
    -d '{"denom":"stake"}' \
    localhost:9090 \
    cosmos.bank.v1beta1.Query/DenomMetadata
```

Example Output:

```json
{
  "metadata": {
    "description": "native staking token of simulation app",
    "denomUnits": [
      {
        "denom": "stake",
        "aliases": [
          "STAKE"
        ]
      }
    ],
    "base": "stake",
    "display": "stake",
    "name": "SimApp Token",
    "symbol": "STK"
  }
}
```

### Query DenomsMetadata

The `DenomsMetadata` endpoint allows users to query metadata for all coin denominations.

```bash
cosmos.bank.v1beta1.Query/DenomsMetadata
```

Example:

```bash
grpcurl -plaintext \
    localhost:9090 \
    cosmos.bank.v1beta1.Query/DenomsMetadata
```

Example Output:

```json
{
  "metadatas": [
    {
      "description": "native staking token of simulation app",
      "denomUnits": [
        {
          "denom": "stake",
          "aliases": [
            "STAKE"
          ]
        }
      ],
      "base": "stake",
      "display": "stake",
      "name": "SimApp Token",
      "symbol": "STK"
    }
  ],
  "pagination": {
    "total": "1"
  }
}
```

### Query DenomOwners

The `DenomOwners` endpoint allows users to query metadata for a single coin denomination.

```bash
cosmos.bank.v1beta1.Query/DenomOwners
```

Example:

```bash
grpcurl -plaintext \
    -d '{"denom":"stake"}' \
    localhost:9090 \
    cosmos.bank.v1beta1.Query/DenomOwners
```

Example Output:

```json
{
  "denomOwners": [
    {
      "address": "cosmos1..",
      "balance": {
        "denom": "stake",
        "amount": "5000000000"
      }
    },
    {
      "address": "cosmos1..",
      "balance": {
        "denom": "stake",
        "amount": "5000000000"
      }
    },
  ],
  "pagination": {
    "total": "2"
  }
}
```

### Query TotalSupply

The `TotalSupply` endpoint allows users to query the total supply of all coins.

```bash
cosmos.bank.v1beta1.Query/TotalSupply
```

Example:

```bash
grpcurl -plaintext \
    localhost:9090 \
    cosmos.bank.v1beta1.Query/TotalSupply
```

Example Output:

```json
{
  "supply": [
    {
      "denom": "stake",
      "amount": "10000000000"
    }
  ],
  "pagination": {
    "total": "1"
  }
}
```

### Query SupplyOf

The `SupplyOf` endpoint allows users to query the total supply of a single coin.

```bash
cosmos.bank.v1beta1.Query/SupplyOf
```

Example:

```bash
grpcurl -plaintext \
    -d '{"denom":"stake"}' \
    localhost:9090 \
    cosmos.bank.v1beta1.Query/SupplyOf
```

Example Output:

```json
{
  "amount": {
    "denom": "stake",
    "amount": "10000000000"
  }
}
```

### Query Params

The `Params` endpoint allows users to query the parameters of the `bank` module.

```bash
cosmos.bank.v1beta1.Query/Params
```

Example:

```bash
grpcurl -plaintext \
    localhost:9090 \
    cosmos.bank.v1beta1.Query/Params
```

Example Output:

```json
{
  "params": {
    "defaultSendEnabled": true
  }
}
```

### Query SendEnabled

The `SendEnabled` enpoints allows users to query the SendEnabled entries of the `bank` module.

Any denominations NOT returned, use the `Params.DefaultSendEnabled` value.

```bash
cosmos.bank.v1beta1.Query/SendEnabled
```

Example:

```bash
grpcurl -plaintext \
    localhost:9090 \
    cosmos.bank.v1beta1.Query/SendEnabled
```

Example Output:

```json
{
  "send_enabled": [
    {
      "denom": "foocoin",
      "enabled": true
    },
    {
      "denom": "barcoin"
    }
  ],
  "pagination": {
    "next-key": null,
    "total": 2
  }
}
```
