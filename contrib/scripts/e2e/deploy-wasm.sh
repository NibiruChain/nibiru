#!/bin/bash
set -e

BINARY="./nibid"
DENOM="unibi"
CHAIN_ID="nibiru-localnet-0"
TXFLAG="--gas-prices 0.1$DENOM --gas auto --gas-adjustment 1.3 -y -b async --chain-id $CHAIN_ID"

# validator addr
VALIDATOR_ADDR=$($BINARY keys show validator --address)
echo "Validator address:"
echo "$VALIDATOR_ADDR"

BALANCE_1=$($BINARY q bank balances "$VALIDATOR_ADDR")
echo "Pre-store balance:"
echo "$BALANCE_1"
echo "TX Flags: $TXFLAG"

TX_HASH=$($BINARY tx wasm store "./contrib/scripts/e2e/contracts/cw_nameservice.wasm" --from validator $TXFLAG --output json | jq -r '.txhash' )
sleep 3

$BINARY q tx $TX_HASH --output json | jq

echo "tx hash: $CONTRACT_CODE"
echo "Stored: $CONTRACT_CODE"

BALANCE_2=$($BINARY q bank balances $VALIDATOR_ADDR)
echo "Post-store balance:"
echo "$BALANCE_2"

INIT='{"purchase_price":{"amount":"100","denom":"unibi"},"transfer_price":{"amount":"999","denom":"unibi"}}'
$BINARY tx wasm instantiate $CONTRACT_CODE "$INIT" --from validator $TXFLAG --label "awesome name service" --no-admin

CONTRACT_ADDRESS=$($BINARY query wasm list-contract-by-code $CONTRACT_CODE --output json | jq -r '.contracts[-1]')
echo "Contract Address: $CONTRACT_ADDRESS"

$BINARY query wasm contract $CONTRACT_ADDRESS

# purchase a domain name
$BINARY tx wasm execute $CONTRACT_ADDRESS '{"register":{"name":"uniques-domain"}}' --amount 100$DENOM --from validator $TXFLAG -y

# query registered name
NAME_QUERY='{"resolve_record": {"name": "uniques-domain"}}'
DOMAIN_OWNER=$($BINARY query wasm contract-state smart $CONTRACT_ADDRESS "$NAME_QUERY" --output json | jq -r '.data.address')
echo "Owner: $DOMAIN_OWNER"

if [ $DOMAIN_OWNER != $VALIDATOR_ADDR ]; then
  echo "Domain owner is not the validator address"
  exit 1
fi