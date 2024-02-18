#!/bin/bash
set -e

# add_genesis_reserve_amt: Used to configure initial reserve values of genesis
# perp markets.
add_genesis_reserve_amt() {
  local M=1000000
  local num_users=300000
  local faucet_nusd_amt=100
  local reserve_amt=$(($num_users * $faucet_nusd_amt * $M))
  echo "$reserve_amt"
}

# add_genesis_perp_markets_with_coingecko_prices: Queries Coingecko to set the
# initial values to for x/perp markets in the genesis.
add_genesis_perp_markets_with_coingecko_prices() {
  local temp_json_fname="tmp_market_prices.json"
  curl -X 'GET' \
    'https://api.coingecko.com/api/v3/simple/price?ids=bitcoin%2Cethereum&vs_currencies=usd' \
    -H 'accept: application/json' \
    >$temp_json_fname

  local reserve_amt=$(add_genesis_reserve_amt)

  price_btc=$(cat tmp_market_prices.json | jq -r '.bitcoin.usd')
  price_btc=${price_btc%.*}
  if [ -z "$price_btc" ]; then
    return 1
  fi

  check_fail() {
    if [ $? -eq 0 ]; then
      echo "Command \"$*\" executed successfully."
    else
      echo "Command \"$*\" failed."
      exit 1
    fi
  }

  nibid genesis add-genesis-perp-market --pair=ubtc:unusd --sqrt-depth="$reserve_amt" --price-multiplier="$price_btc" --oracle-pair="ubtc:uusd"
  check_fail nibid genesis add-genesis-perp-market

  price_eth=$(cat tmp_market_prices.json | jq -r '.ethereum.usd')
  price_eth=${price_eth%.*}
  if [ -z "$price_eth" ]; then
    return 1
  fi

  nibid genesis add-genesis-perp-market --pair=ueth:unusd --sqrt-depth=$reserve_amt --price-multiplier="$price_eth" --oracle-pair="ueth:uusd"
  check_fail nibid genesis add-genesis-perp-market

  echo 'tmp_market_prices: '
  cat $temp_json_fname | jq .
  rm -f $temp_json_fname
}

add_genesis_perp_markets_offline() {
  local reserve_amt=$(add_genesis_reserve_amt)
  price_btc="20000"
  price_eth="2000"
  nibid genesis add-genesis-perp-market --pair=ubtc:unusd --sqrt-depth=$reserve_amt --price-multiplier=$price_btc
  nibid genesis add-genesis-perp-market --pair=ueth:unusd --sqrt-depth=$reserve_amt --price-multiplier=$price_eth
}

