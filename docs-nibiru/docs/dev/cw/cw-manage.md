---
order: 4
---
# Managing Smart Contracts

Compile your CosmWasm smart contract to a .wasm file using Rust,
optimize it, and store it on the chain. Track the transaction hash.
Instantiate the contract for execution, then query and execute actions
using the provided commands.{synopsis}

## Compile

After creating your cosmwasm smart contract in rust, you will
need to first compile and generate a wasm binary executable file `.wasm`.

Ensure that your `.cargo/config` contains:

```rust
wasm = "build --release --target wasm32-unknown-unknown"
```

Then compile your contract using:

```bash
cargo wasm
```

Finally, we need to optimize our generated wasm binary file using CosmWasm Rust
Optimizer by running:

```bash
docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/optimizer:0.15.0 
```

To specify a single contract, you can add it's path at the end of the docker command

```bash
docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/optimizer:0.15.0 <path-to-contract>
```

The resulting artifact should be in the default directory under your project `artifact/<contract-name>.wasm`.

## Store

Next step for bringing your contract to live is storing it onto the chain:

```bash
FROM=<your-wallet-address>

nibid tx wasm store artifacts/<contract-name>.wasm \
    --from $FROM \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices 0.025unibi \
    --yes
```

This will return a transaction json result, we need to keep track of the transaction hash so it is better to run:

```bash
FROM=<your-wallet-address>

TXHASH="$(nibid tx wasm store artifacts/contract.wasm \
    --from $FROM \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices 0.025unibi \
    --yes | jq -rcs '.[0].txhash')"
```

Now we have the transaction hash stored under `$TXHASH`. Next we need to obtain the
contract's `code id` and store it in `$CODE_ID`.

Query the transaction hash:

```bash
nibid q tx $TXHASH > txhash.json
```

Save the `CODE_ID` for later usage:

```bash
CODE_ID="$(cat txhash.json | jq -r '.logs[0].events[1].attributes[1].value')"
```

To check which wallet is currently setup, run

```bash
nibid keys show -a wallet
```

## Instantiate

To instantiate your contract, we need to know of the required instantiate message.
We can store it locally for future usage:

```bash
echo '{"some_msg": {}}' | jq . | tee instantiate_args.json
```

Now that we gave the `$FROM` address, the contract `$CODE_ID` and the instantiate json message.
We can instantiate by running:

```bash
nibid tx wasm instantiate $CODE_ID \
  "$(cat instantiate_args.json)" \
  --admin "$FROM" \
  --label <contract-label> \
  --from "$FROM" \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.025unibi \
  --yes
```

Let's now obtain the contract address:

```bash
TXHASH_INIT="$(nibid tx wasm instantiate $CODE_ID \
    "$(cat instantiate_args.json)" \
    --admin "$FROM" \
    --label <contract-label> \
    --from $FROM \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices 0.025unibi \
    --yes | jq -rcs '.[0].txhash')"
```

Query the transaction:

```bash
nibid q tx $TXHASH_INIT > txhash.init.json
```

Save the contract address:

```bash
CONTRACT_ADDRESS="$(cat txhash.init.json | jq -r '.logs[0].events[1].attributes[0].value')"
```

## Query

In order to query your contract, you can run the following:

```bash
nibid query wasm contract-state smart $CONTRACT_ADDRESS '<query-message>'
```

You can also use a local file liek in the example below:

```bash
echo '{"some_query": {}}' | jq . | tee query_args.json
```

```bash
nibid query wasm contract-state smart $CONTRACT_ADDRESS "$(cat ./query_args.json) | jq
```

## Execute

To execute, we can run the below. Let's keep track of the transaction hash to verify it
on the [Nibiru's explorer](https://nibiru.explorers.guru/).

```bash
echo '{"some_exec": {}}' | jq . | tee exec_args.json
```

```bash
nibid tx wasm execute $CONTRACT_ADDRESS \                                                   ✘ INT
  "$(cat ./exec_args.json)" \
  --from $FROM \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.025unibi | jq
```

You will need to approve the transaction and spend the required gas.

After running the transaction, you can utilize the transaction hash to view any events and any attributes for the contract.

```bash
TXHASH_EXECUTE="$(nibid tx wasm execute $CONTRACT_ADDRESS \ 
      "$(cat ./exec_args.json)" \
      --from $FROM \
      --gas auto \
      --gas-adjustment 1.5 \
      --gas-prices 0.025unibi
      -yes | jq -rcs '.[0].txhash')"
```

```bash
nibid q tz $TXHASH_EXECUTE | jq
```

## Related Pages

- [Nibiru EVM](../../evm/README.md)
- [Nibiru Networks](../networks)
- [Nibiru Source Code Installation Guide](../cli/nibid-binary.md#install-option-3--building-from-the-source-code)
