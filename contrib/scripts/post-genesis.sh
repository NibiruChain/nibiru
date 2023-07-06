#!/bin/sh
set -e

BINARY=nibid
SUDO_ACCOUNT="validator"

# Deploy contract
# $1: contract flag
# $2: contract path
# $3: contract from
deploy_contract() {
  CONTRACT_FLAG=$1
  CONTRACT_PATH=$2
  CONTRACT_FROM=$3

  $BINARY tx wasm store "$CONTRACT_PATH" --from "$CONTRACT_FROM" --gas 6000000
  $BINARY tx wasm instantiate 1 '{}' --from "$CONTRACT_FROM" --label "$CONTRACT_FLAG" --no-admin
}