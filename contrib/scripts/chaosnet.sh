#!/bin/sh
set -e

# Set localnet settings
MNEMONIC=${MNEMONIC:-"guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"}
CHAIN_ID=${CHAIN_ID:-"nibiru-localnet-0"}

rm -rf $HOME/.nibid
nibid init $CHAIN_ID --chain-id $CHAIN_ID --home $HOME/.nibid --overwrite
nibid config keyring-backend test
nibid config chain-id $CHAIN_ID
nibid config broadcast-mode block
nibid config output json

sed -i '/\[api\]/,+3 s/enable = false/enable = true/' $HOME/.nibid/config/app.toml
sed -i 's/swagger = false/swagger = true/' $HOME/.nibid/config/app.toml
sed -i 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' $HOME/.nibid/config/app.toml
sed -i 's/127.0.0.1/0.0.0.0/' $HOME/.nibid/config/config.toml
echo "$MNEMONIC" | nibid keys add validator --recover
nibid genesis add-genesis-account $(nibid keys show validator -a) "10000000000000unibi,10000000000000unusd,10000000000000uusdt,10000000000000uusdc"
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

  local num_users=24000
  local faucet_nusd_amt=100
  local quote_amt=$(($num_users * $faucet_nusd_amt * $M))

  price_btc=$(cat tmp_market_prices.json | jq -r '.bitcoin.usd')
  price_btc=${price_btc%.*}
  base_amt_btc=$(($quote_amt / $price_btc))

  price_eth=$(cat tmp_market_prices.json | jq -r '.ethereum.usd')
  price_eth=${price_eth%.*}
  base_amt_eth=$(($quote_amt / $price_eth))

  nibid add-genesis-perp-market --pair=ubtc:unusd --base-amt=$base_amt_btc --quote-amt=$quote_amt --max-leverage=12
  nibid add-genesis-perp-market --pair=ueth:unusd --base-amt=$base_amt_eth --quote-amt=$quote_amt --max-leverage=20 --mmr=0.04

  echo 'tmp_market_prices: '
  cat $temp_json_fname | jq .
  rm -f $temp_json_fname
}

add_genesis_perp_markets_default() {
  # nibid add-genesis-perp-market [pair] [base-asset-reserve] [quote-asset-reserve] [trade-limit-ratio] [fluctuation-limit-ratio] [maxOracle-spread-ratio] [maintenance-margin-ratio] [max-leverage]
  local KILO="000"
  local MEGA="000000"
  local quote_amt=10$KILO$MEGA
  local base_amt_btc=$(($quote_amt / 16500))
  local base_amt_eth=$(($quote_amt / 1200))
  nibid add-genesis-perp-market --pair=ubtc:unusd --base-amt=$base_amt_btc --quote-amt=$quote_amt --max-leverage=12
  nibid add-genesis-perp-market --pair=ueth:unusd --base-amt=$base_amt_eth --quote-amt=$quote_amt --max-leverage=20 --mmr=0.04
}

add_genesis_perp_markets_with_coingecko_prices

# x/perp
add_genesis_param '.app_state.perp.params.stopped = false'
add_genesis_param '.app_state.perp.params.fee_pool_fee_ratio = "0.001"'
add_genesis_param '.app_state.perp.params.ecosystem_fund_fee_ratio = "0.001"'
add_genesis_param '.app_state.perp.params.liquidation_fee_ratio = "0.025"'
add_genesis_param '.app_state.perp.params.partial_liquidation_ratio = "0.25"'
add_genesis_param '.app_state.perp.params.funding_rate_interval = "30 min"'
add_genesis_param '.app_state.perp.params.twap_lookback_window = "900s"'
add_genesis_param '.app_state.perp.pair_metadata[0].pair = "ubtc:unusd"'
add_genesis_param '.app_state.perp.pair_metadata[0].latest_cumulative_premium_fraction = "0"'
add_genesis_param '.app_state.perp.pair_metadata[1].pair = "ueth:unusd"'
add_genesis_param '.app_state.perp.pair_metadata[1].latest_cumulative_premium_fraction = "0"'

# x/oracle
add_genesis_param '.app_state.oracle.params.twap_lookback_window = "900s"'
add_genesis_param '.app_state.oracle.params.vote_period = "10"'
add_genesis_param '.app_state.oracle.params.min_voters = "1"'
add_genesis_param '.app_state.oracle.exchange_rates[0].pair = "ubtc:unusd"'
add_genesis_param '.app_state.oracle.exchange_rates[0].exchange_rate = "20000"'
add_genesis_param '.app_state.oracle.exchange_rates[1].pair = "ueth:unusd"'
add_genesis_param '.app_state.oracle.exchange_rates[1].exchange_rate = "2000"'
