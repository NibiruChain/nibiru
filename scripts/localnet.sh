#!/bin/sh
#set -e

# Console log text colour
red=`tput setaf 9`
green=`tput setaf 10`
blue=`tput setaf 12`
reset=`tput sgr0`

echo_info () {
  echo "${blue}"
  echo "$1"
  echo "${reset}"
}

echo_error () {
  echo "${red}"
  echo "$1"
  echo "${reset}"
}

echo_success () {
  echo "${green}"
  echo "$1"
  echo "${reset}"
}


echo_info "Building from source..."
if make build; then 
  echo_success "Successfully built binary"
else
  echo_error "Could not build binary. Failed to make build"
fi

# Set localnet settings
BINARY=./build/matrixd
CHAIN_ID=localnet
CHAIN_DIR=./data
RPC_PORT=26657
GRPC_PORT=9090
MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
GENESIS_COINS=1000000000stake,1000000000validatortoken

# Stop matrixd if it is already running
if pgrep -x "$BINARY" >/dev/null; then
    echo_error "Terminating $BINARY..."
    killall matrixd
fi

# Remove previous data
echo_info "Removing previous chain data from $CHAIN_DIR/$CHAIN_ID..."
rm -rf $CHAIN_DIR/$CHAIN_ID

# Add directory for chain, exit if error
if ! mkdir -p $CHAIN_DIR/$CHAIN_ID 2>/dev/null; then
    echo_error "Failed to create chain folder. Aborting..."
    exit 1
fi

# Initialize matrixd with "localnet" chain id
echo_info "Resetting database $CHAIN_ID..."
if $BINARY --home $CHAIN_DIR/$CHAIN_ID unsafe-reset-all; then
  echo_success "Successfully reset database"
else
  echo_error "Failed to reset database"
fi


# Initialize matrixd with "localnet" chain id
echo_info "Initializing $CHAIN_ID..."
if $BINARY --home $CHAIN_DIR/$CHAIN_ID init test --chain-id=$CHAIN_ID; then
  echo_success "Successfully initialized $CHAIN_ID"
else
  echo_error "Failed to initialize $CHAIN_ID"
fi

echo_info "Adding genesis accounts..."
echo "$MNEMONIC" | $BINARY --home $CHAIN_DIR/$CHAIN_ID keys add validator --recover --keyring-backend=test
$BINARY --home $CHAIN_DIR/$CHAIN_ID add-genesis-account $($BINARY --home $CHAIN_DIR/$CHAIN_ID keys show validator --keyring-backend test -a) $GENESIS_COINS

echo_info "Adding gentx validator..."
$BINARY --home $CHAIN_DIR/$CHAIN_ID gentx validator 1000000000stake --chain-id $CHAIN_ID --keyring-backend=test

echo_info "Collecting gentx..."
if $BINARY --home $CHAIN_DIR/$CHAIN_ID collect-gentxs; then
  echo_success "Successfully collected genesis txs into genesis.json"
else
  echo_error "Failed to collect genesis txs"
fi

# Start the network
echo_info "Starting $CHAIN_ID in $CHAIN_DIR..."
echo_info "Log file is located at $CHAIN_DIR/$CHAIN_ID.log"
$BINARY --home $CHAIN_DIR/$CHAIN_ID start --pruning=nothing --grpc.address="0.0.0.0:$GRPC_PORT"