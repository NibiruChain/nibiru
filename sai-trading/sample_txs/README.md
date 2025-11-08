# Sample transactions

This folder holds sample transactions that can be used to test the transaction.

These are multi-message json transactions that have to be signed and sent to a
running chain.

## Chain set-up

To run the transactions, you need to have a running chain. You can use the
localnet chain of Nibiru Chain ([instruction here](https://github.com/NibiruChain/nibiru?tab=readme-ov-file#running-a-local-node)).

We will need to deploy the contracts that are used in the transactions, namely:

- a `vault` contract to act as counterparty for traders
- a `vault_token_minter` contract to mint tokens for the vault
- a `perp` contract to act as the perpetual contract and main interface for traders
- a `oracle` contract to provide price feeds to the perpetual contract

### Instructions

The contracts wasm can be found in the `artifact` folder of this repository.
We will do admin function using our famous `guardcream` validator, which already holds a lot of funds on the localnet chain.

```bash
nibid keys add validator --keyring-backend test --recover
> guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host
```

We'll also use our tx pipe to make it easy to grab results from transactions:

```bash
alias tx='jq -rcs ".[0].txhash" | { read txhash; sleep 6; nibid q tx $txhash | jq }'
alias get_code='jq -r '\''.events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value'\'
alias get_contract_address='jq -r '\''.events[] | select(.type == "instantiate") | .attributes[] | select(.key == "_contract_address") | .value'\'''
```

We can also set-up the common transaction flag we will use accross this tutorial.

```bash
TX_FLAG=(--fees 750000unibi --gas 30000000 --yes --keyring-backend test --chain-id nibiru-localnet-0)
```

#### Deploy the contracts

We will first store the contracts:

```bash
cd artifacts
PERP_CODE_ID=$(nibid tx wasm store perp.wasm --from validator "${TX_FLAG[@]}" | tx | get_code)
VAULT_CODE_ID=$(nibid tx wasm store vault.wasm --from validator "${TX_FLAG[@]}" | tx | get_code)
VAULT_TOKEN_MINTER_CODE_ID=$(nibid tx wasm store vault_token_minter.wasm --from validator "${TX_FLAG[@]}" | tx | get_code)
ORACLE_CODE_ID=$(nibid tx wasm store oracle.wasm --from validator "${TX_FLAG[@]}" | tx | get_code)

echo "PERP_CODE_ID: $PERP_CODE_ID\nVAULT_CODE_ID: $VAULT_CODE_ID\nVAULT_TOKEN_MINTER_CODE_ID: $VAULT_TOKEN_MINTER_CODE_ID\nORACLE_CODE_ID: $ORACLE_CODE_ID"
```

And then we can instantiate them. We can go to the `sample_txs` folder and levearge the init messages in the `init_msg` folder.

```bash
cd ../sample_txs

ORACLE_ADDRESS=$(nibid tx wasm instantiate $ORACLE_CODE_ID "$(cat instantiate_messages/oracle.json)" --amount 1000unibi --label "oracle" --admin nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl --from validator "${TX_FLAG[@]}" | tx | get_contract_address)

# update the init messages with the oracle address for the vault and perp contracts
jq --arg oracle "$ORACLE_ADDRESS" '.oracle_address = $oracle' instantiate_messages/perp.json > tmp.json && mv tmp.json instantiate_messages/perp.json
jq --arg oracle "$ORACLE_ADDRESS" '.oracle = $oracle' instantiate_messages/vault.json > tmp.json && mv tmp.json instantiate_messages/vault.json

PERP_ADDRESS=$(nibid tx wasm instantiate $PERP_CODE_ID "$(cat instantiate_messages/perp.json)" --amount 1000unibi --label "perp" --admin nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl --from validator "${TX_FLAG[@]}" | tx | get_contract_address)


VAULT_TOKEN_MINTER=$(nibid tx wasm instantiate $VAULT_TOKEN_MINTER_CODE_ID "$(cat instantiate_messages/vault_token_minter.json)" --amount 1000unibi --label "vault_token_minter" --admin nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl --from validator "${TX_FLAG[@]}" | tx | get_contract_address)

# update the vault init message with the perp address
jq --arg perp_contract "$PERP_ADDRESS" '.perp_contract = $perp_contract' instantiate_messages/vault.json > tmp.json && mv tmp.json instantiate_messages/vault.json

# update the vault init message with the vault token minter address
jq --arg vault_token_minter_contract "$VAULT_TOKEN_MINTER" '.vault_token_minter_contract = $vault_token_minter_contract' instantiate_messages/vault.json > tmp.json && mv tmp.json instantiate_messages/vault.json

VAULT_ADDRESS=$(nibid tx wasm instantiate $VAULT_CODE_ID "$(cat instantiate_messages/vault.json)" --amount 1000unibi --label "vault" --admin nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl --from validator "${TX_FLAG[@]}" | tx | get_contract_address)

echo "PERP_ADDRESS: $PERP_ADDRESS\nVAULT_ADDRESS: $VAULT_ADDRESS\nVAULT_TOKEN_MINTER: $VAULT_TOKEN_MINTER\nORACLE_ADDRESS: $ORACLE_ADDRESS"
```

## Market set-up

Before being able to trade, we will need to set-up things like markets, parameters, oracle prices etc. We will use our messages defined in setup_market.json to do so.

This will:

- Update Oracle Address
- Update Vault
- Update Collaterals
- Update Trading Activated
- Set Pairs
- Set Groups
- Set Fees
- Update Pair Depths
- Update Oi Windows settings
- Update Borrowing Pairs
- Update Borrowing Groups
- Update Borrowing Pair ois
- Update Borrowing Group ois
- Update Fee Tiers

- Post prices for the main pairs

```bash
# update the contract addresses in the setup_market.json file
jq --arg PERP_ADDRESS "$PERP_ADDRESS" '.body.messages[].contract = $PERP_ADDRESS' setup_market.json > setup_market_temp.json && mv setup_market_temp.json setup_market.json

nibid tx sign setup_market.json --from validator ${TX_FLAG[@]} | jq | tee signed.json
nibid tx broadcast signed.json --from validator ${TX_FLAG[@]} | tx | jq .raw_log

# Post prices for the main pairs
jq --arg ORACLE_ADDRESS "$ORACLE_ADDRESS" '.body.messages[].contract = $ORACLE_ADDRESS' setup_prices.json > setup_prices_temp.json && mv setup_prices_temp.json setup_prices.json
nibid tx sign setup_prices.json --from validator ${TX_FLAG[@]} | jq | tee signed.json
nibid tx broadcast signed.json --from validator ${TX_FLAG[@]} | tx | jq .raw_log
```

## Open and close positions

Now we are ready to open a position.

```bash
nibid tx wasm execute $PERP_ADDRESS "$(cat open_trade.json)" --from validator ${TX_FLAG[@]} --amount 1000unusd | tx | jq .raw_log
```
