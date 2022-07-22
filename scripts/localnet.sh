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
if make install; then
  echo_success "Successfully built binary"
else
  echo_error "Could not build binary. Failed to make install."
  exit 1
fi

# Set localnet settings
BINARY=nibid
CHAIN_ID=nibiru-localnet-0
CHAIN_HOME_DIR=$HOME/.nibid
RPC_PORT=26657
GRPC_PORT=9090
MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
GENESIS_COINS=1000000000stake,1000000000validatortoken,1000000000unibi,10000000000000unusd

# Stop nibid if it is already running
if pgrep -x "$BINARY" >/dev/null; then
    echo_error "Terminating $BINARY..."
    killall nibid
fi

# Remove previous data
echo_info "Removing previous chain data from $CHAIN_HOME_DIR..."
rm -rf $CHAIN_HOME_DIR

# Add directory for chain, exit if error
if ! mkdir -p $CHAIN_HOME_DIR 2>/dev/null; then
  echo_error "Failed to create chain folder. Aborting..."
  exit 1
fi

# Initialize nibid with "localnet" chain id
echo_info "Initializing $CHAIN_ID..."
if $BINARY init nibiru-localnet-0 --home $CHAIN_HOME_DIR --chain-id $CHAIN_ID; then
  echo_success "Successfully initialized $CHAIN_ID"
else
  echo_error "Failed to initialize $CHAIN_ID"
fi


# Configure keyring-backend to "test"
echo_info "Configuring keyring-backend..."
if $BINARY config keyring-backend test --home $CHAIN_HOME_DIR; then
  echo_success "Successfully configured keyring-backend"
else
  echo_error "Failed to configure keyring-backend"
fi


# Configure chain-id
echo_info "Configuring chain-id..."
if $BINARY config chain-id $CHAIN_ID --home $CHAIN_HOME_DIR; then
  echo_success "Successfully configured chain-id"
else
  echo_error "Failed to configure chain-id"
fi

# Configure broadcast mode
echo_info "Configuring broadcast mode..."
if $BINARY config broadcast-mode block --home $CHAIN_HOME_DIR; then
  echo_success "Successfully configured broadcast-mode"
else
  echo_error "Failed to configure broadcast mode"
fi

# Configure output mode
echo_info "Configuring output mode..."
if $BINARY config output json --home $CHAIN_HOME_DIR; then
  echo_success "Successfully configured output mode"
else
  echo_error "Failed to configure output mode"
fi

# Enable API Server
echo_info "Enabling API server"
if sed -i '' '/\[api\]/,+3 s/enable = false/enable = true/' $CHAIN_HOME_DIR/config/app.toml; then
  echo_success "Successfully enabled API server"
else
  echo_error "Failed to enable API server"
fi

# Enable Swagger Docs
echo_info "Enabling Swagger Docs"
if sed -i '' 's/swagger = false/swagger = true/' $CHAIN_HOME_DIR/config/app.toml; then
  echo_success "Successfully enabled Swagger Docs"
else
  echo_error "Failed to enable Swagger Docs"
fi

# Enable CORS for localnet
echo_info "Enabling CORS"
if sed -i '' 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' $CHAIN_HOME_DIR/config/app.toml; then
  echo_success "Successfully enabled CORS"
else
  echo_error "Failed to enable CORS"
fi

echo_info "Adding genesis accounts..."
echo "$MNEMONIC" | $BINARY keys add validator --recover --home $CHAIN_HOME_DIR
if $BINARY add-genesis-account $($BINARY keys show validator -a --home $CHAIN_HOME_DIR) $GENESIS_COINS --home $CHAIN_HOME_DIR; then
  echo_success "Successfully added genesis accounts"
else
  echo_error "Failed to add genesis accounts"
fi

echo_info "Adding gentx validator..."
if $BINARY gentx validator 900000000stake --home $CHAIN_HOME_DIR --chain-id $CHAIN_ID; then
  echo_success "Successfully added gentx"
else
  echo_error "Failed to add gentx"
fi

echo_info "Collecting gentx..."
if $BINARY --home $CHAIN_HOME_DIR collect-gentxs; then
  echo_success "Successfully collected genesis txs into genesis.json"
else
  echo_error "Failed to collect genesis txs"
fi

# Start the network
echo_info "Starting $CHAIN_ID in $CHAIN_HOME_DIR..."
$BINARY start --home $CHAIN_HOME_DIR
