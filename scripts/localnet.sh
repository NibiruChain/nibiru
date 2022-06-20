#!/bin/sh
#set -e

# Console log text colour
red=$(tput setaf 9)
green=$(tput setaf 10)
blue=$(tput setaf 12)
reset=$(tput sgr0)

echo_info() {
  echo "${blue}"
  echo "$1"
  echo "${reset}"
}

echo_error() {
  echo "${red}"
  echo "$1"
  echo "${reset}"
}

echo_success() {
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
BINARY=./build/nibid
CHAIN_ID=nibiru-localnet-0
CHAIN_DIR=./data
RPC_PORT=26657
GRPC_PORT=9090
VALIDATOR_MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
SHRIMP_MNEMONIC="item segment elevator fork swim tone search hope enough asthma apology pact embody extra trash educate deposit raccoon giant gift essay able female develop"
WHALE_MNEMONIC="throw oblige vague twist clutch grunt physical sell conduct blossom owner delay suspect square kidney joy define book boss outside reason silk success you"
LIQUIDATOR_MNEMONIC="oxygen tattoo pond upgrade barely sudden wheat correct dumb roast glance conduct scene profit female health speak hire north grab allow provide depend away"
ORACLE_MNEMONIC="abandon wave reason april rival valid saddle cargo aspect toe tomato stomach zero quick side potato artwork mixture over basket sort churn palace cherry"
GENESIS_COINS=1000000000stake,10000000000000000000unibi,10000000000000000000unusd

# Stop nibid if it is already running
if pgrep -x "$BINARY" >/dev/null; then
  echo_error "Terminating $BINARY..."
  killall nibid
fi

# Remove previous data
echo_info "Removing previous chain data from $CHAIN_DIR..."
rm -rf $CHAIN_DIR

# Add directory for chain, exit if error
if ! mkdir -p $CHAIN_DIR 2>/dev/null; then
  echo_error "Failed to create chain folder. Aborting..."
  exit 1
fi

# Initialize nibid with "localnet" chain id
echo_info "Initializing $CHAIN_ID..."
if $BINARY init nibiru-localnet-0 --home $CHAIN_DIR --chain-id $CHAIN_ID; then
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

# Configure broadcast mode
echo_info "Configuring broadcast mode..."
if $BINARY config broadcast-mode block --home $CHAIN_DIR; then
  echo_success "Successfully configured broadcast-mode"
else
  echo_error "Failed to configure broadcast mode"
fi

# Enable API Server
echo_info "Enabling API server"
if sed -i '' '/\[api\]/,+3 s/enable = false/enable = true/' $CHAIN_DIR/config/app.toml; then
  echo_success "Successfully enabled API server"
else
  echo_error "Failed to enable API server"
fi

# Enable Swagger Docs
echo_info "Enabling Swagger Docs"
if sed -i '' 's/swagger = false/swagger = true/' $CHAIN_DIR/config/app.toml; then
  echo_success "Successfully enabled Swagger Docs"
else
  echo_error "Failed to enable Swagger Docs"
fi

# Enable CORS for localnet
echo_info "Enabling CORS"
if sed -i '' 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' $CHAIN_DIR/config/app.toml; then
  echo_success "Successfully enabled CORS"
else
  echo_error "Failed to enable CORS"
fi

echo_info "Adding genesis accounts..."

# validator
echo "$VALIDATOR_MNEMONIC" | $BINARY keys add validator --recover --home $CHAIN_DIR
$BINARY add-genesis-account $($BINARY keys show validator -a --home $CHAIN_DIR) $GENESIS_COINS --home $CHAIN_DIR

# shrimp
echo "$SHRIMP_MNEMONIC" | $BINARY keys add shrimp --recover --home $CHAIN_DIR
$BINARY add-genesis-account $($BINARY keys show shrimp -a --home $CHAIN_DIR) $GENESIS_COINS --home $CHAIN_DIR

# whale
echo "$WHALE_MNEMONIC" | $BINARY keys add whale --recover --home $CHAIN_DIR
$BINARY add-genesis-account $($BINARY keys show whale -a --home $CHAIN_DIR) $GENESIS_COINS --home $CHAIN_DIR

# liquidator
echo "$LIQUIDATOR_MNEMONIC" | $BINARY keys add liquidator --home $CHAIN_DIR --recover
$BINARY add-genesis-account $($BINARY keys show liquidator -a --home $CHAIN_DIR) $GENESIS_COINS --home $CHAIN_DIR

# oracle
echo "$ORACLE_MNEMONIC" | $BINARY keys add oracle --home $CHAIN_DIR --recover --keyring-backend test
$BINARY add-genesis-account $($BINARY keys show oracle -a --home $CHAIN_DIR --keyring-backend test) $GENESIS_COINS --home $CHAIN_DIR

echo_success "Genesis accounts added"

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
$BINARY start --home $CHAIN_DIR --log_level info
