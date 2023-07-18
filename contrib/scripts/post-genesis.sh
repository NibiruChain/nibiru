#!/bin/sh
set -e

OUTPUT_DIR=build/contracts

### Helper functions

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

build_contract_tx() {
  if [ $# -ne 2 ]
  then
    echo "Incorrect number of arguments. Expected 2, got $#."
    display_help
    exit 1
  fi

  path=$1
  output=$2

  nibid tx wasm store "$path" --generate-only > "$output"
  echo "Building contract txs... $path"
}

display_help() {
    echo "Usage: $0 <command>"
    echo "Commands:"
    echo "  build <contract_path> <output_file>: build contract txs"
    echo "  help: print this help message"
    exit 0
}

# Entry point

# take command line arguments
command=$1

# parse command line arguments
case $command in
  "build")
    build_contract_tx "$2"
    ;;
  "help")
    display_help
    ;;
  *)
    echo "Unknown command: $command"
    exit 1
    ;;
esac

BINARY=nibid
SUDO_ACCOUNT="validator"
