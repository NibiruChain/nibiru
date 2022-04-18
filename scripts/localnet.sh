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
  exit 1
fi

# Set localnet settings
BINARY=./build/matrixd
CHAIN_ID=nibiru-test-chain-0
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
echo_info "Removing previous chain data from $CHAIN_DIR..."
rm -rf $CHAIN_DIR

# Add directory for chain, exit if error
if ! mkdir -p $CHAIN_DIR 2>/dev/null; then
  echo_error "Failed to create chain folder. Aborting..."
  exit 1
fi

# Initialize matrixd with "localnet" chain id
echo_info "Initializing $CHAIN_ID..."
if $BINARY init matrix-test-chain-0 --home $CHAIN_DIR --chain-id $CHAIN_ID; then
  echo_success "Successfully initialized $CHAIN_ID"
else
  echo_error "Failed to initialize $CHAIN_ID"
fi


# Configure keyring-backend to "test"
echo_info "Configuring keyring-backend..."
if $BINARY config keyring-backend test --home $CHAIN_DIR; then
  echo_success "Successfully configured keyring-backend"
else
  echo_error "Failed to configure keyring-backend"
fi


# Configure chain-id
echo_info "Configuring chain-id..."
if $BINARY config chain-id $CHAIN_ID --home $CHAIN_DIR; then
  echo_success "Successfully configured chain-id"
else
  echo_error "Failed to configure chain-id"
fi


echo_info "Adding genesis accounts..."
echo "$MNEMONIC" | $BINARY keys add validator --recover --home $CHAIN_DIR
if $BINARY add-genesis-account $($BINARY keys show validator -a --home $CHAIN_DIR) $GENESIS_COINS --home $CHAIN_DIR; then
  echo_success "Successfully added genesis accounts"
else
  echo_error "Failed to add genesis accounts"
fi

echo_info "Adding gentx validator..."
if $BINARY gentx validator 900000000stake --home $CHAIN_DIR --chain-id $CHAIN_ID; then
  echo_success "Successfully added gentx"
else
  echo_error "Failed to add gentx"
fi

echo_info "Collecting gentx..."
if $BINARY --home $CHAIN_DIR collect-gentxs; then
  echo_success "Successfully collected genesis txs into genesis.json"
else
  echo_error "Failed to collect genesis txs"
fi

# Start the network
echo_info "Starting $CHAIN_ID in $CHAIN_DIR..."
$BINARY start --home $CHAIN_DIR
