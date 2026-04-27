#!/usr/bin/env bash
#
# This script is used in tandem with `contrib/docker/chaosnet.Dockerfile` to
# run nodes for Nibiru Chain networks inside docker containers.
#
# See CHAOSNET.md for usage instructions.
#
# How chaosnet.sh works:
# - Parameterizes env vars for Docker Compose multi-node use.
# - Edits bind addresses to 0.0.0.0 so services are reachable across containers.

set -e

log_error() {
  echo "ERROR: $*" >&2
}

which_ok() {
  if which "$1" >/dev/null 2>&1; then
    return 0
  else
    log_error "$1 is not present in \$PATH"
    return 1
  fi
}

# sed_inplace: Edits files in place across GNU and BSD sed variants.
# macOS/BSD sed requires an explicit backup suffix after -i; '' means none.
sed_inplace() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}

# Preflight checks for dependencies
which_ok jq
which_ok sed
which_ok nibid

# Set localnet settings
DEFAULT_MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
MNEMONIC=${MNEMONIC:-"$DEFAULT_MNEMONIC"}
CHAIN_ID=${CHAIN_ID:-"nibiru-localnet-0"}
CHAIN_DIR=${CHAIN_DIR:-"$HOME/.nibid"}
LCD_PORT=${LCD_PORT:-"1317"}
GRPC_PORT=${GRPC_PORT:-"9090"}
RPC_PORT=${RPC_PORT:-"26657"}

if [[ -d "$CHAIN_DIR" ]]; then
  # Preserve keyring/config while clearing chain data and generated genesis files.
  nibid tendermint unsafe-reset-all --home "$CHAIN_DIR"
  rm -f "$CHAIN_DIR/config/genesis.json"
  rm -rf "$CHAIN_DIR/config/gentx/"
fi

mkdir -p "$CHAIN_DIR"
nibid init "$CHAIN_ID" --chain-id "$CHAIN_ID" --home "$CHAIN_DIR" --overwrite
nibid config keyring-backend test --home "$CHAIN_DIR"
nibid config chain-id "$CHAIN_ID" --home "$CHAIN_DIR"
nibid config broadcast-mode sync --home "$CHAIN_DIR"
nibid config output json --home "$CHAIN_DIR"

sed_inplace "s/127.0.0.1:26657/0.0.0.0:$RPC_PORT/" "$CHAIN_DIR/config/config.toml"
sed_inplace 's/log_format = .*/log_format = "json"/' "$CHAIN_DIR/config/config.toml"

sed_inplace '/\[api\]/,+3 s/enable = false/enable = true/' "$CHAIN_DIR/config/app.toml"
sed_inplace 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$CHAIN_DIR/config/app.toml"
sed_inplace "s/localhost:1317/0.0.0.0:$LCD_PORT/" "$CHAIN_DIR/config/app.toml"
sed_inplace '/\[grpc\]/,+3 s/enable = false/enable = true/' "$CHAIN_DIR/config/app.toml"
sed_inplace "s/localhost:9090/0.0.0.0:$GRPC_PORT/" "$CHAIN_DIR/config/app.toml"

# ------------------------------------------------------------------------
# Configure genesis params
# ------------------------------------------------------------------------

# add_genesis_params runs a jq command to edit fields of the genesis.json .
#
# Args:
#   $1 : the jq input that gets mapped to the json.
add_genesis_param() {
  echo "jq input $1"
  local gen_json tmp_gen_json
  gen_json="$CHAIN_DIR/config/genesis.json"
  tmp_gen_json="$CHAIN_DIR/config/tmp_genesis.json"
  # copy param ($1) to tmp_genesis.json
  jq "$1" "$gen_json" > "$tmp_gen_json"
  # rewrite genesis.json with the contents of tmp_genesis.json
  mv "$tmp_gen_json" "$gen_json"
}

# recover mnemonic
key_name="validator"

recover_validator_key() {
  echo "$MNEMONIC" | nibid keys add "$key_name" --recover --home "$CHAIN_DIR"
}

validate_mnemonic() {
  local words
  read -r -a words <<< "$MNEMONIC"
  if [[ ${#words[@]} -ne 24 ]]; then
    log_error "MNEMONIC must contain 24 words; got ${#words[@]}"
    exit 1
  fi
}

# Validator key handling:
# - missing key: recover it from MNEMONIC, whether default or custom
# - existing key with default MNEMONIC: keep it for idempotent local reruns
# - existing key with custom MNEMONIC: replace it so genesis uses that mnemonic
if nibid keys show "$key_name" --home "$CHAIN_DIR" >/dev/null 2>&1; then
  if [[ "$MNEMONIC" != "$DEFAULT_MNEMONIC" ]]; then
    validate_mnemonic
    echo "Replacing existing validator key because MNEMONIC is non-default"
    nibid keys delete "$key_name" --home "$CHAIN_DIR" -y
    recover_validator_key
  fi
else
  recover_validator_key
fi

val_address="$(nibid keys show "$key_name" -a --home "$CHAIN_DIR")"
nibid genesis add-genesis-account "$val_address" "10000000000000unibi" --home "$CHAIN_DIR"

# x/sudo
add_genesis_param ".app_state.sudo.sudoers.root = \"$val_address\""

# ------------------------------------------------------------------------
# genesis accounts and balances
# ------------------------------------------------------------------------
nibid genesis add-genesis-account nibi1wx9360p9rvy9m5cdhsua6qpdf9ktvwhjqw949s "10000000000000unibi" --home "$CHAIN_DIR" # faucet
nibid genesis add-genesis-account nibi1g7vzqfthhf4l4vs6skyjj27vqhe97m5gp33hxy "10000000000000unibi" --home "$CHAIN_DIR" # liquidator

# gen_txs
nibid genesis gentx "$key_name" 900000000unibi --chain-id "$CHAIN_ID" --home "$CHAIN_DIR"
nibid genesis collect-gentxs --home "$CHAIN_DIR"
