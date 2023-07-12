#!/bin/sh
set -e

# Set localnet settings
MNEMONIC=${MNEMONIC:-"guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"}
CHAIN_ID=${CHAIN_ID:-"nibiru-localnet-0"}

rm -rf $HOME/.nibid
nibid init $CHAIN_ID --chain-id $CHAIN_ID --home $HOME/.nibid --overwrite
nibid config keyring-backend test
nibid config chain-id $CHAIN_ID
nibid config broadcast-mode sync
nibid config output json

sed -i '/\[api\]/,+3 s/enable = false/enable = true/' $HOME/.nibid/config/app.toml
sed -i 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' $HOME/.nibid/config/app.toml
sed -i 's/127.0.0.1/0.0.0.0/' $HOME/.nibid/config/config.toml
sed -i 's/localhost/0.0.0.0/' $HOME/.nibid/config/config.toml
sed -i 's/localhost/0.0.0.0/' $HOME/.nibid/config/app.toml
sed -i 's/localhost/0.0.0.0/' $HOME/.nibid/config/app.toml

echo "$MNEMONIC" | nibid keys add validator --recover
nibid genesis add-genesis-account $(nibid keys show validator -a) "10000000000000unibi,10000000000000unusd,10000000000000uusdt,10000000000000uusdc"
nibid genesis add-genesis-account nibi1wx9360p9rvy9m5cdhsua6qpdf9ktvwhjqw949s "10000000000000unibi,10000000000000unusd,10000000000000uusdt,10000000000000uusdc" # faucet
nibid genesis add-genesis-account nibi1g7vzqfthhf4l4vs6skyjj27vqhe97m5gp33hxy "10000000000000unibi" # liquidator
nibid genesis add-genesis-account nibi19n0clnacpjv0d3t8evvzp3fptlup9srjdqunzs "10000000000000unibi" # pricefeeder
nibid genesis gentx validator 900000000unibi --chain-id $CHAIN_ID
nibid genesis collect-gentxs

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


add_genesis_perp_markets_with_coingecko_prices() {
  local temp_json_fname="tmp_market_prices.json"
  curl -X 'GET' \
    'https://api.coingecko.com/api/v3/simple/price?ids=bitcoin%2Cethereum&vs_currencies=usd' \
    -H 'accept: application/json' \
    >$temp_json_fname

  local M=1000000

  local num_users=300000
  local faucet_nusd_amt=100
  local reserve_amt=$(($num_users * $faucet_nusd_amt * $M))

  price_btc=$(cat tmp_market_prices.json | jq -r '.bitcoin.usd')
  price_btc=${price_btc%.*}
  if [ -z "$price_btc" ]; then
    return 1
  fi

  nibid genesis add-genesis-perp-market --pair=ubtc:unusd --sqrt-depth=$reserve_amt --price-multiplier=$price_btc

  price_eth=$(cat tmp_market_prices.json | jq -r '.ethereum.usd')
  price_eth=${price_eth%.*}
  if [ -z "$price_eth" ]; then
    return 1
  fi

  nibid genesis add-genesis-perp-market --pair=ueth:unusd --sqrt-depth=$reserve_amt --price-multiplier=$price_eth
}

add_genesis_perp_markets_with_coingecko_prices

# x/oracle
add_genesis_param '.app_state.oracle.params.twap_lookback_window = "900s"'
add_genesis_param '.app_state.oracle.params.vote_period = "10"'
add_genesis_param '.app_state.oracle.params.min_voters = "1"'

nibid genesis add-genesis-pricefeeder-delegation --validator $(nibid keys show validator -a --bech val) --pricefeeder nibi19n0clnacpjv0d3t8evvzp3fptlup9srjdqunzs