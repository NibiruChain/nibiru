#!/usr/bin/env bash
#
# This script is used in tandem with `contrib/docker/chaosnet.Dockerfile` to
# run nodes for Nibiru Chain networks inside docker containers. 
# - See CHAOSNET.md for usage instructions.
set -e

# Set localnet settings
MNEMONIC=${MNEMONIC:-"guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"}
CHAIN_ID=${CHAIN_ID:-"nibiru-localnet-0"}
LCD_PORT=${LCD_PORT:-"1317"}
GRPC_PORT=${GRPC_PORT:-"9090"}
RPC_PORT=${RPC_PORT:-"26657"}

rm -rf "$HOME/.nibid"
nibid init $CHAIN_ID --chain-id $CHAIN_ID --home $HOME/.nibid --overwrite
nibid config keyring-backend test
nibid config chain-id $CHAIN_ID
nibid config broadcast-mode sync
nibid config output json

sed -i "s/127.0.0.1:26657/0.0.0.0:$RPC_PORT/" $HOME/.nibid/config/config.toml
sed -i 's/log_format = .*/log_format = "json"/' $HOME/.nibid/config/config.toml

sed -i '/\[api\]/,+3 s/enable = false/enable = true/' $HOME/.nibid/config/app.toml
sed -i 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' $HOME/.nibid/config/app.toml
sed -i "s/localhost:1317/0.0.0.0:$LCD_PORT/" $HOME/.nibid/config/app.toml
sed -i "s/localhost:9090/0.0.0.0:$GRPC_PORT/" $HOME/.nibid/config/app.toml


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
  cat $HOME/.nibid/config/genesis.json | jq "$1" >$HOME/.nibid/config/tmp_genesis.json
  # rewrite genesis.json with the contents of tmp_genesis.json
  mv $HOME/.nibid/config/tmp_genesis.json $HOME/.nibid/config/genesis.json
}

curr_dir="$(dirname "$0")"
source "$curr_dir/feat-perp.sh"
add_genesis_perp_markets_offline

# recover mnemonic
echo "$MNEMONIC" | nibid keys add validator --recover
nibid genesis add-genesis-account $(nibid keys show validator -a) "10000000000000unibi"

# x/sudo
add_genesis_param ".app_state.sudo.sudoers.root = \"$(nibid keys show validator -a)\""

# x/oracle
add_genesis_param '.app_state.oracle.params.twap_lookback_window = "900s"'
add_genesis_param '.app_state.oracle.params.vote_period = "10"'
add_genesis_param '.app_state.oracle.params.min_voters = "1"'
nibid genesis add-genesis-pricefeeder-delegation --validator $(nibid keys show validator -a --bech val) --pricefeeder nibi19n0clnacpjv0d3t8evvzp3fptlup9srjdqunzs

# ------------------------------------------------------------------------
# genesis accounts and balances
# ------------------------------------------------------------------------
nibid genesis add-genesis-account nibi1wx9360p9rvy9m5cdhsua6qpdf9ktvwhjqw949s "10000000000000unibi" # faucet
nibid genesis add-genesis-account nibi1g7vzqfthhf4l4vs6skyjj27vqhe97m5gp33hxy "10000000000000unibi" # liquidator
nibid genesis add-genesis-account nibi19n0clnacpjv0d3t8evvzp3fptlup9srjdqunzs "10000000000000unibi" # pricefeeder

# gen_txs
nibid genesis gentx validator 900000000unibi --chain-id $CHAIN_ID
nibid genesis collect-gentxs
