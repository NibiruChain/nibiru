#!/bin/sh
set -e

# Console log text colour
console_log_text_color() {
  red=$(tput setaf 9)
  green=$(tput setaf 10)
  blue=$(tput setaf 12)
  reset=$(tput sgr0)
}

if [ console_log_text_color ]; then echo "succesfully toggled console coloring"
else
  # For Ubuntu and Debian. MacOS has tput by default.
  apt-get install libncurses5-dbg -y
fi

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
BINARY="nibid"
CHAIN_ID="nibiru-localnet-0"
RPC_PORT="26657"
GRPC_PORT="9090"
MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
GENESIS_COINS="1000000000unibi,10000000000000unusd"
CHAIN_DIR="$HOME/.nibid"

# Stop nibid if it is already running
if pgrep -x "$BINARY" > /dev/null; then
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
if $BINARY init nibiru-localnet-0 --chain-id $CHAIN_ID --overwrite; then
  echo_success "Successfully initialized $CHAIN_ID"
else
  echo_error "Failed to initialize $CHAIN_ID"
fi


# Configure keyring-backend to "test"
echo_info "Configuring keyring-backend..."
if $BINARY config keyring-backend test; then
  echo_success "Successfully configured keyring-backend"
else
  echo_error "Failed to configure keyring-backend"
fi


# Configure chain-id
echo_info "Configuring chain-id..."
if $BINARY config chain-id $CHAIN_ID; then
  echo_success "Successfully configured chain-id"
else
  echo_error "Failed to configure chain-id"
fi

# Configure broadcast mode
echo_info "Configuring broadcast mode..."
if $BINARY config broadcast-mode block; then
  echo_success "Successfully configured broadcast-mode"
else
  echo_error "Failed to configure broadcast mode"
fi

# Configure output mode
echo_info "Configuring output mode..."
if $BINARY config output json; then
  echo_success "Successfully configured output mode"
else
  echo_error "Failed to configure output mode"
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
echo "$MNEMONIC" | $BINARY keys add validator --recover
if $BINARY add-genesis-account $($BINARY keys show validator -a) $GENESIS_COINS; then
  echo_success "Successfully added genesis accounts"
else
  echo_error "Failed to add genesis accounts"
fi

echo_info "Adding gentx validator..."
if $BINARY gentx validator 900000000unibi --chain-id $CHAIN_ID; then
  echo_success "Successfully added gentx"
else
  echo_error "Failed to add gentx"
fi

echo_info "Collecting gentx..."
if $BINARY collect-gentxs; then
  echo_success "Successfully collected genesis txs into genesis.json"
else
  echo_error "Failed to collect genesis txs"
fi

# ------------------------------------------------------------------------
# Configure genesis params
# ------------------------------------------------------------------------

# add_genesis_params runs a jq command to edit fields of the genesis.json .
#
# Args:
#   $1 : the jq input that gets mapped to the json.
add_genesis_param() {
  echo "jq input $1"
  # copy param ($1) to tmp_genesis.json
  cat $CHAIN_DIR/config/genesis.json | jq "$1" > $CHAIN_DIR/config/tmp_genesis.json
  # rewrite genesis.json with the contents of tmp_genesis.json
  mv $CHAIN_DIR/config/tmp_genesis.json $CHAIN_DIR/config/genesis.json
}

echo_info "Configuring genesis params"

# x/vpool
# nibid add-genesis-vpool [pair] [base-asset-reserve] [quote-asset-reserve] [trade-limit-ratio] [fluctuation-limit-ratio] [maxOracle-spread-ratio] [maintenance-margin-ratio] [max-leverage]
# BTC:NUSD
nibid add-genesis-vpool ubtc:unusd 50000000000 1000000000000000 0.1 0.1 0.1 0.0625 12

# ETH:NUSD
nibid add-genesis-vpool ueth:unusd 666666000000 1000000000000000 0.1 0.1 0.1 0.0625 10

# x/perp
add_genesis_param '.app_state.perp.params.stopped = false'
add_genesis_param '.app_state.perp.params.fee_pool_fee_ratio = "0.001"'
add_genesis_param '.app_state.perp.params.ecosystem_fund_fee_ratio = "0.001"'
add_genesis_param '.app_state.perp.params.liquidation_fee_ratio = "0.025"'
add_genesis_param '.app_state.perp.params.partial_liquidation_ratio = "0.25"'
add_genesis_param '.app_state.perp.params.funding_rate_interval = "30 min"'
add_genesis_param '.app_state.perp.params.twap_lookback_window = "900s"'
add_genesis_param '.app_state.perp.pair_metadata[0].pair = {token0:"ubtc",token1:"unusd"}'
add_genesis_param '.app_state.perp.pair_metadata[0].cumulative_premium_fractions = ["0"]'
add_genesis_param '.app_state.perp.pair_metadata[1].pair = {token0:"ueth",token1:"unusd"}'
add_genesis_param '.app_state.perp.pair_metadata[1].cumulative_premium_fractions = ["0"]'

# x/pricefeed
nibid add-genesis-oracle nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl

cat $HOME/.nibid/config/genesis.json | jq '.app_state.pricefeed.params.twap_lookback_window = "900s"' > $HOME/.nibid/config/tmp_genesis.json && mv $HOME/.nibid/config/tmp_genesis.json $HOME/.nibid/config/genesis.json

# Start the network
echo_info "Starting $CHAIN_ID in $CHAIN_DIR..."
$BINARY start
