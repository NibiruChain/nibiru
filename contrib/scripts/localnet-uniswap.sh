#!/bin/bash
set -e

# Set localnet settings
BINARY="nibid"
CHAIN_ID="nibiru-localnet-0"
MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
CHAIN_DIR="$HOME/.nibid"
GENESIS_EXPORT="./contrib/genesis.json"

echo "CHAIN_DIR: $CHAIN_DIR"
echo "CHAIN_ID: $CHAIN_ID"

if ! command -v "$BINARY" >/dev/null 2>&1; then
  >&2 echo "Error: Binary '$BINARY' not found in PATH. Build or install it first."
  exit 1
fi
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
  echo "succesfully toggled console coloring"
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

# $FLAG_SKIP_BUILD: toggles whether to build from source. The default
#   behavior of the script is to run make install if the flag --no-build is omitted.
FLAG_SKIP_BUILD=false

build_from_source() {
  echo_info "Building from source..."
  if make install; then
    echo_success "Successfully built binary"
  else
    echo_error "Could not build binary. Failed to make install."
    exit 1
  fi
}

# enable_feature_flag: Enables feature flags variables if present
enable_feature_flag() {
  case $1 in
  spot) FLAG_SPOT=true ;;
  *) echo_error "Unknown feature: $1" ;;
  esac
}

# Iterate over flags, handling the cases: "--no-build" and "--features"
while [[ $# -gt 0 ]]; do
  case $1 in
  --no-build)
    FLAG_SKIP_BUILD=true
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


# Check if FLAG_SKIP_BUILD was set to true
if ! $FLAG_SKIP_BUILD; then
  build_from_source
fi

echo_info "Features flags:"
echo "FLAG_SKIP_BUILD: $FLAG_SKIP_BUILD"

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
  pkill -x "$BINARY" || true
fi

# Remove previous data, preserving keyring and config files
echo_info "Removing previous chain data from $CHAIN_DIR..."
$BINARY tendermint unsafe-reset-all
rm -f "$CHAIN_DIR/config/genesis.json"
rm -rf "$CHAIN_DIR/config/gentx/"

# Add directory for chain, exit if error
if ! mkdir -p "$CHAIN_DIR" 2>/dev/null; then
  echo_error "Failed to create chain folder. Aborting..."
  exit 1
fi

# 1. Initialize chain
if "$BINARY" init "$CHAIN_ID" --chain-id "$CHAIN_ID" --overwrite --home "$CHAIN_DIR"; then
  echo_success "Successfully initialized $CHAIN_ID"
else
  echo_error "Failed to initialize $CHAIN_ID"
  exit 1
fi

# 2. Replace exported genesis
if [ ! -f "$GENESIS_EXPORT" ]; then
    echo "Error: exported genesis file $GENESIS_EXPORT not found!"
    exit 1
fi

cp "$GENESIS_EXPORT" "$CHAIN_DIR/config/genesis.json"

# 3. Update chain_id in genesis
sed -i "s/\"chain_id\": \".*\"/\"chain_id\": \"$CHAIN_ID\"/" "$CHAIN_DIR/config/genesis.json"

# 4. Reset old state
nibid tendermint unsafe-reset-all --home "$CHAIN_DIR"

# 5. (Optional) Create a new validator key and add to genesis
if ! "$BINARY" keys show validator --home "$CHAIN_DIR" >/dev/null 2>&1; then
    "$BINARY" keys add validator --home "$CHAIN_DIR"
    "$BINARY" add-genesis-account validator "$STAKE" --home "$CHAIN_DIR"
    "$BINARY" gentx validator "$STAKE" --chain-id "$CHAIN_ID" --home "$CHAIN_DIR"
    "$BINARY" collect-gentxs --home "$CHAIN_DIR"
fi

# 6. Start the new chain
echo "=== Starting chain $CHAIN_ID ==="
nibid start --home "$CHAIN_DIR"
