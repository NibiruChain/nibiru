#!/bin/bash
set -e

# Set localnet settings
BINARY="nibid"
CHAIN_ID="nibiru-localnet-0"
MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
GENESIS_COINS="10000000000000unibi,10000000000000unusd,10000000000000uusdt,10000000000000uusdc"
CHAIN_DIR="$HOME/.nibid"

echo "CHAIN_DIR: $CHAIN_DIR"
echo "CHAIN_ID: $CHAIN_ID"

# ------------------------------------------------------------
# Set up colored text logging
# ------------------------------------------------------------

# Console log text colour
console_log_text_color() {
  red=$(tput setaf 9)
  green=$(tput setaf 10)
  blue=$(tput setaf 12)
  reset=$(tput sgr0)
}

if [ console_log_text_color ]; then
  echo "successfully toggled console coloring"
else
  # For Ubuntu and Debian. MacOS has tput by default.
  apt-get install libncurses5-dbg -y
fi

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

# ------------------------------------------------------------
# Flag parsing
# ------------------------------------------------------------

echo_info "Parsing flags for the script..."

# $FLAG_NO_BUILD: toggles whether to build from source. The default
#   behavior of the script is to run make install if the flag --no-build is not present.
FLAG_NO_BUILD=false

# $FLAG_PERP: Feature flag for x/perp. Enabled with `--features perp`.
FLAG_PERP=false

# $FLAG_SPOT: Feature flag for x/spot. Enabled with `--features spot`.
FLAG_SPOT=false

build_from_source() {
  echo_info "Building from source..."
  if make install; then
    echo_success "Successfully built binary"
  else
    echo_error "Could not build binary. Failed to make install."
    exit 1
  fi
}

# Initialize an associative array for feature flags with default values
declare -A features=( ["perp"]=0 ["spot"]=0 )

# enable_feature_flag: Enables feature flags variables if present
enable_feature_flag() {
  case $1 in
    perp) FLAG_PERP=true ;;
    spot) FLAG_SPOT=true ;;
    *) echo_error "Unknown feature: $1" ;;
  esac
}

# Iterate over flags, handling the cases: "--no-build" and "--features"
while [[ $# -gt 0 ]]; do
  case $1 in
    --no-build)
      FLAG_NO_BUILD=true
      shift
      ;;
    --features)
      shift # Remove '--features' from arguments
      while [[ $# -gt 0 && $1 != --* ]]; do
        enable_feature_flag "$1"
        shift # Remove the feature name from arguments
      done
      ;;
    *) shift ;; # Unknown arg
  esac
done

# Check if FLAG_NO_BUILD was set to true
if ! $FLAG_NO_BUILD; then
  build_from_source
fi

echo_info "Features flags:"
echo "FLAG_NO_BUILD: $FLAG_NO_BUILD"
echo "FLAG_PERP: $FLAG_PERP"
echo "FLAG_SPOT: $FLAG_SPOT"

SEDOPTION=""
if [[ "$OSTYPE" == "darwin"* ]]; then
  SEDOPTION="''"
fi

# ------------------------------------------------------------------------
echo_info "Successfully finished localnet script setup."
# ------------------------------------------------------------------------

# Stop nibid if it is already running
if pgrep -x "$BINARY" >/dev/null; then
  echo_error "Terminating $BINARY..."
  killall nibid
fi

# Remove previous data
echo_info "Removing previous chain data from $CHAIN_DIR..."
rm -rf "$CHAIN_DIR"

# Add directory for chain, exit if error
if ! mkdir -p "$CHAIN_DIR" 2>/dev/null; then
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

# nibid config
echo_info "Updating nibid config..."
$BINARY config keyring-backend test
$BINARY config chain-id $CHAIN_ID
$BINARY config broadcast-mode sync
$BINARY config output json
$BINARY config # Prints config.

# Enable API Server
echo_info "config/app.toml: Enabling API server"
sed -i $SEDOPTION '/\[api\]/,+3 s/enable = false/enable = true/' $CHAIN_DIR/config/app.toml

# Enable Swagger Docs
echo_info "config/app.toml: Enabling Swagger Docs"
sed -i $SEDOPTION 's/swagger = false/swagger = true/' $CHAIN_DIR/config/app.toml

# Enable CORS for localnet
echo_info "config/app.toml: Enabling CORS"
sed -i $SEDOPTION 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' $CHAIN_DIR/config/app.toml

echo_info "Adding genesis accounts..."

val_key_name="validator"

echo "$MNEMONIC" | $BINARY keys add $val_key_name --recover
$BINARY add-genesis-account $($BINARY keys show $val_key_name -a) $GENESIS_COINS
echo_success "Successfully added genesis account: $val_key_name"

val_address=$($BINARY keys list | jq -r '.[] | select(.name == "validator") | .address')
val_address=${val_address:-"nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"}

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
  cat $CHAIN_DIR/config/genesis.json | jq "$1" >$CHAIN_DIR/config/tmp_genesis.json
  # rewrite genesis.json with the contents of tmp_genesis.json
  mv $CHAIN_DIR/config/tmp_genesis.json $CHAIN_DIR/config/genesis.json
}

echo_info "Configuring genesis params"

if $FLAG_PERP; then
  source "./feat-perp.sh"

  if add_genesis_perp_markets_with_coingecko_prices; then
    echo_success "set perp markets with coingecko prices"
  elif add_genesis_perp_markets_offline; then
    echo_success "set perp markets with offline defaults"
  else
    echo_error "failed to set genesis perp markets"
    exit 1
  fi
fi

# if $FLAG_SPOT; then
#   # Perform any actions specific to the x/spot feature
# fi

# set validator as sudoer
add_genesis_param '.app_state.sudo.sudoers.root = "'"$val_address"'"'

# hack for localnet since we don't have a pricefeeder yet
add_genesis_param '.app_state.oracle.exchange_rates[0].pair = "ubtc:uuusd"'
add_genesis_param '.app_state.oracle.exchange_rates[0].exchange_rate = "'"$price_btc"'"'
add_genesis_param '.app_state.oracle.exchange_rates[1].pair = "ueth:uuusd"'
add_genesis_param '.app_state.oracle.exchange_rates[1].exchange_rate = "'"$price_eth"'"'

add_genesis_param '.app_state.inflation.params.inflation_enabled = false'

# ------------------------------------------------------------------------
# Gentx
# ------------------------------------------------------------------------

echo_info "Adding gentx validator..."
if $BINARY genesis gentx $val_key_name 900000000unibi --chain-id $CHAIN_ID; then
  echo_success "Successfully added gentx"
else
  echo_error "Failed to add gentx"
fi

echo_info "Collecting gentx..."
if $BINARY genesis collect-gentxs; then
  echo_success "Successfully collected genesis txs into genesis.json"
else
  echo_error "Failed to collect genesis txs"
fi

# ------------------------------------------------------------------------
# Start the network
# ------------------------------------------------------------------------

echo_info "Starting $CHAIN_ID in $CHAIN_DIR..."
$BINARY start --home "$CHAIN_DIR" --pruning nothing
