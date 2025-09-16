---
order: 6
---

# Creating Fungible Tokens Guide

The `nibi-stargate` smart contract exemplifies the creation and management of fungible tokens on the Nibiru Chain. This guide presents steps for engaging with the smart contract, encompassing environment setup, contract deployment, transaction broadcasting, and message type comprehension.

## Clone nibiru-wasm

Start by cloning the `nibi-stargate` contract from the `nibiru-wasm` [repository](https://github.com/NibiruChain/nibiru-wasm/tree/main).

```bash
git clone https://github.com/NibiruChain/nibiru-wasm.git
```

To build and verify select contracts, you can use the [`just` comand runner](https://github.com/casey/just?tab=readme-ov-file#prerequisites).

```bash
just
```

The specific `nibi-stargate` contract is located under `contracts/`.

## Setting Environment Variables

Ensure the `nibid` command-line interface is installed before interacting with the nibi-stargate smart contract. Refer to the `nibid` installation [guide](../cli/nibid-binary.md).

The following environment variables are utilized in this guide:

- `KEYNAME`: The keyring name or wallet address for the account responsible for broadcasting transactions. It defaults to "validator" in local Nibiru instances.
- `tx alias`: This alias facilitates reading transaction responses. It extracts the transaction hash using jq, waits for 3 seconds to ensure transaction processing, queries transaction details using nibid q tx, and appends structured information to out.json.

```bash
KEYNAME="validator" # wallet address
alias tx="jq -rcs '.[0].txhash' | { read txhash; sleep 3; nibid q tx \$txhash | jq '{txhash, height, code, logs, tx, gas_wanted, gas_used}' >> out.json}"
```

## Deploying the Smart Contract

To deploy the smart contract, you need a reference to its stored bytecode on the blockchain. If the bytecode is not yet stored, deploy or store it:

```bash
nibid tx wasm store ./artifacts/nibi_stargate.wasm \
    --from $KEYNAME \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices 0.025unibi \
    --yes | jq -rcs '.[0].txhash'
```

This will return the transaction hash (tx-hash) with which we will query against to identify the `CODE ID`:

```bash
nibid q tx <tx-hash> | jq -r '.logs[0].events[1].attributes[1].value'
```

Then, instantiate the contract using the stored `CODE ID` and it will return with the transaction hash which we will use to the new contract address.
Note: The `InstantiateMsg` for this smart contract is empty `({})`.

```bash
CODE_ID=<CODE ID>

nibid tx wasm instantiate $CODE_ID {} \
    --admin "$KEYNAME" \
    --label "fungible tokens" \
    --from "$KEYNAME" \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices 0.025unibi \
    --yes | jq -rcs '.[0].txhash'
```

Using the transaction hash (tx-hash),

```bash
nibid q tx <tx-hash> | jq -r '.logs[0].events[1].attributes[0].value'
```

### Broadcasting Transactions

The `nibi-stargate` smart contract defines the following `ExecuteMsg` enum for broadcasting transactions:

```rust
pub enum ExecuteMsg {
    CreateDenom { subdenom: String },
    Mint { coin: Coin, mint_to: String },
    Burn { coin: Coin, burn_from: String },
    ChangeAdmin { denom: String, new_admin: String },
}
```

#### CreateDenom

```json
{ "create_denom": { "subdenom": "zzz" } }
```

#### Mint

```json
{ 
    "mint": { 
        "coin": { "amount": "[amount]", "denom": "tf/[contract-addr]/[subdenom]" }, 
        "mint_to": "[mint-to-addr]" 
    } 
}
```

#### Burn

```json
{ 
    "burn": { 
        "coin": { "amount": "[amount]", "denom": "tf/[contract-addr]/[subdenom]" }, 
        "burn_from": "[burn-from-addr]" 
    } 
}
```

#### ChangeAdmin

```json
{ 
    "change_admin": { 
        "denom": "tf/[contract-addr]/[subdenom]", 
        "new_admin": "[ADDR]" 
    } 
}
```

### Executing Broadcast Messages

Using the known Contract Address and the example broadcast messages above, you can execute them via the below command. (`broadcast_msg.json` refers to the json file containing the message)

```bash
CONTRACT_ADDRESS=<contract-address>

nibid tx wasm execute $CONTRACT_ADDRESS \
    "$(cat broadcast_msg.json)" \
    --from $FROM \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices 0.025unibi \
    --yes | jq
```
