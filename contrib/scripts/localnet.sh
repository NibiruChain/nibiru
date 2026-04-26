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
  killall nibid
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

# Initialize nibid with "localnet" chain id
echo_info "Initializing $CHAIN_ID..."
if $BINARY init $CHAIN_ID --chain-id $CHAIN_ID --overwrite; then
  echo_success "Successfully initialized $CHAIN_ID"
else
  echo_error "Failed to initialize $CHAIN_ID"
  exit 1
fi

# nibid config
echo_info "Updating nibid config..."
$BINARY config keyring-backend test
$BINARY config chain-id $CHAIN_ID
$BINARY config broadcast-mode sync
$BINARY config output json
$BINARY config node "http://localhost:26657"
$BINARY config # Prints config.

# Enable API Server
echo_info "config/app.toml: Enabling API server"
sed -i $SEDOPTION '/\[api\]/,+3 s/enable = false/enable = true/' $CHAIN_DIR/config/app.toml

# Enable GRPC Server
echo_info "config/app.toml: Enabling GRPC server"
sed -i $SEDOPTION '/\[grpc\]/,+3 s/enable = false/enable = true/' $CHAIN_DIR/config/app.toml


# Enable JSON RPC Server
echo_info "config/app.toml: Enabling JSON API server"
sed -i $SEDOPTION '/\[json\-rpc\]/,+3 s/enable = false/enable = true/' $CHAIN_DIR/config/app.toml

echo_info "config/app.toml: Enabling debug evm api"
sed -i $SEDOPTION '/\[json\-rpc\]/,+13 s/api = "eth,net,web3"/api = "eth,net,web3,debug"/' $CHAIN_DIR/config/app.toml

echo_info "config/app.toml: Enabling evm indexer"
sed -i $SEDOPTION '/\[json\-rpc\]/,+51 s/enable-indexer = false/enable-indexer = true/' $CHAIN_DIR/config/app.toml

# Enable Swagger Docs
echo_info "config/app.toml: Enabling Swagger Docs"
sed -i $SEDOPTION 's/swagger = false/swagger = true/' $CHAIN_DIR/config/app.toml

# Enable CORS for localnet
echo_info "config/app.toml: Enabling CORS"
sed -i $SEDOPTION 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' $CHAIN_DIR/config/app.toml

echo_info "Adding genesis accounts..."

val_key_name="validator"

if ! $BINARY keys show $val_key_name; then 
  echo "$MNEMONIC" | $BINARY keys add $val_key_name --recover
  echo_success "Successfully added key: $val_key_name"
fi

val_address=$($BINARY keys show $val_key_name -a)
val_address=${val_address:-"nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"}

$BINARY genesis add-genesis-account $val_address $GENESIS_COINS
# EVM encoded nibi address for the same account
$BINARY genesis add-genesis-account nibi1cr6tg4cjvux00pj6zjqkh6d0jzg7mksaywxyl3 $GENESIS_COINS
$BINARY genesis add-genesis-account nibi1ltez0kkshywzm675rkh8rj2eaf8et78cqjqrhc $GENESIS_COINS
echo_success "Successfully added genesis account: $val_key_name"

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

add_genesis_param_slurpfile() {
  local jq_input="$1"
  local slurp_var="$2"
  local json_file="$3"
  echo "jq input $jq_input (slurpfile: $slurp_var <- $json_file)"
  jq --slurpfile "$slurp_var" "$json_file" "$jq_input" \
    "$CHAIN_DIR/config/genesis.json" >"$CHAIN_DIR/config/tmp_genesis.json"
  mv "$CHAIN_DIR/config/tmp_genesis.json" "$CHAIN_DIR/config/genesis.json"
}

echo_info "Configuring genesis params"

# set validator as sudoer
add_genesis_param '.app_state.sudo.sudoers.root = "'"$val_address"'"'

# Constants inherited from mainnet
MAINNET_WNIBI_ADDR="0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97"
MAINNET_WNIBI_BECH32="nibi1pjk0v60cg347e2pxjyarc6uk4n2tq25hhymgg9"
MAINNET_WNIBI_CODE_HASH="0xffb88e0eb48147949565e65de3ec8a54b746214da7b9dd5b9a8a3ae7df46193b"

# NOTE: This only affects fresh localnet bootstraps because it mutates genesis.
# If nibid is already running from an older genesis, rerun this script from
# scratch to make WNIBI live at the canonical address.
#
# Localnet normally has the canonical WNIBI param set, but no contract actually
# deployed at that address. Seed both auth and EVM genesis so code paths that
# special-case canonical WNIBI behave the same way they do on mainnet.
#
# This mirrors the canonical WNIBI contract data embedded in the v2.7.0 upgrade
# helper, so localnet uses the same address and initial state layout.
# Make WNIBI live on localnet at the canonical mainnet address so local tooling
# can rely on the same contract address and storage layout as mainnet.
add_genesis_param '.app_state.evm.params.canonical_wnibi = "'"$MAINNET_WNIBI_ADDR"'"'
add_genesis_param '.app_state.auth.accounts += [{
  "@type": "/eth.types.v1.EthAccount",
  "base_account": {
    "address": "'"$MAINNET_WNIBI_BECH32"'",
    "pub_key": null,
    # Pick the next open account number in genesis at patch time.
    "account_number": ((.app_state.auth.accounts | length) | tostring),
    "sequence": "0"
  },
  "code_hash": "'"$MAINNET_WNIBI_CODE_HASH"'"
}]'

# The auth module also needs a matching EthAccount entry for the contract
# address. The EVM module stores bytecode/storage separately, but genesis import
# expects the auth-side contract account to exist with the correct code hash.
#
# Keep the WNIBI genesis payload as JSON and slurp it into jq instead of
# inlining a giant escaped string. This payload mirrors the canonical WNIBI
# contract data used by the v2.7.0 upgrade helper.
WNIBI_EVM_GENESIS_JSON="$(mktemp)"
trap 'rm -f "$WNIBI_EVM_GENESIS_JSON"' EXIT
cat >"$WNIBI_EVM_GENESIS_JSON" <<'EOF'
{
  "address": "0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97",
  "code": "6080604052600436106100a05760003560e01c8063313ce56711610064578063313ce567146101b257806370a08231146101dd57806395d89b411461021a578063a9059cbb14610245578063d0e30db014610282578063dd62ed3e1461028c576100af565b806306fdde03146100b9578063095ea7b3146100e457806318160ddd1461012157806323b872dd1461014c5780632e1a7d4d14610189576100af565b366100af576100ad6102c9565b005b6100b76102c9565b005b3480156100c557600080fd5b506100ce61036f565b6040516100db9190610b18565b60405180910390f35b3480156100f057600080fd5b5061010b60048036038101906101069190610bd3565b6103fd565b6040516101189190610c2e565b60405180910390f35b34801561012d57600080fd5b506101366104ef565b6040516101439190610c58565b60405180910390f35b34801561015857600080fd5b50610173600480360381019061016e9190610c73565b6104f7565b6040516101809190610c2e565b60405180910390f35b34801561019557600080fd5b506101b060048036038101906101ab9190610cc6565b61085b565b005b3480156101be57600080fd5b506101c7610995565b6040516101d49190610d0f565b60405180910390f35b3480156101e957600080fd5b5061020460048036038101906101ff9190610d2a565b6109a8565b6040516102119190610c58565b60405180910390f35b34801561022657600080fd5b5061022f6109c0565b60405161023c9190610b18565b60405180910390f35b34801561025157600080fd5b5061026c60048036038101906102679190610bd3565b610a4e565b6040516102799190610c2e565b60405180910390f35b61028a6102c9565b005b34801561029857600080fd5b506102b360048036038101906102ae9190610d57565b610a63565b6040516102c09190610c58565b60405180910390f35b34600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282546103189190610dc6565b925050819055503373ffffffffffffffffffffffffffffffffffffffff167fe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c346040516103659190610c58565b60405180910390a2565b6000805461037c90610e29565b80601f01602080910402602001604051908101604052809291908181526020018280546103a890610e29565b80156103f55780601f106103ca576101008083540402835291602001916103f5565b820191906000526020600020905b8154815290600101906020018083116103d857829003601f168201915b505050505081565b600081600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040516104dd9190610c58565b60405180910390a36001905092915050565b600047905090565b600081600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054101561054557600080fd5b3373ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff161415801561061d57507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205414155b1561073f5781600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410156106ab57600080fd5b81600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282546107379190610e5a565b925050819055505b81600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825461078e9190610e5a565b9250508190555081600360008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282546107e49190610dc6565b925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040516108489190610c58565b60405180910390a3600190509392505050565b80600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410156108a757600080fd5b80600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282546108f69190610e5a565b925050819055503373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f19350505050158015610943573d6000803e3d6000fd5b503373ffffffffffffffffffffffffffffffffffffffff167f7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b658260405161098a9190610c58565b60405180910390a250565b600260009054906101000a900460ff1681565b60036020528060005260406000206000915090505481565b600180546109cd90610e29565b80601f01602080910402602001604051908101604052809291908181526020018280546109f990610e29565b8015610a465780601f10610a1b57610100808354040283529160200191610a46565b820191906000526020600020905b815481529060010190602001808311610a2957829003601f168201915b505050505081565b6000610a5b3384846104f7565b905092915050565b6004602052816000526040600020602052806000526040600020600091509150505481565b600081519050919050565b600082825260208201905092915050565b60005b83811015610ac2578082015181840152602081019050610aa7565b60008484015250505050565b6000601f19601f8301169050919050565b6000610aea82610a88565b610af48185610a93565b9350610b04818560208601610aa4565b610b0d81610ace565b840191505092915050565b60006020820190508181036000830152610b328184610adf565b905092915050565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000610b6a82610b3f565b9050919050565b610b7a81610b5f565b8114610b8557600080fd5b50565b600081359050610b9781610b71565b92915050565b6000819050919050565b610bb081610b9d565b8114610bbb57600080fd5b50565b600081359050610bcd81610ba7565b92915050565b60008060408385031215610bea57610be9610b3a565b5b6000610bf885828601610b88565b9250506020610c0985828601610bbe565b9150509250929050565b60008115159050919050565b610c2881610c13565b82525050565b6000602082019050610c436000830184610c1f565b92915050565b610c5281610b9d565b82525050565b6000602082019050610c6d6000830184610c49565b92915050565b600080600060608486031215610c8c57610c8b610b3a565b5b6000610c9a86828701610b88565b9350506020610cab86828701610b88565b9250506040610cbc86828701610bbe565b9150509250925092565b600060208284031215610cdc57610cdb610b3a565b5b6000610cea84828501610bbe565b91505092915050565b600060ff82169050919050565b610d0981610cf3565b82525050565b6000602082019050610d246000830184610d00565b92915050565b600060208284031215610d4057610d3f610b3a565b5b6000610d4e84828501610b88565b91505092915050565b60008060408385031215610d6e57610d6d610b3a565b5b6000610d7c85828601610b88565b9250506020610d8d85828601610b88565b9150509250929050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000610dd182610b9d565b9150610ddc83610b9d565b9250828201905080821115610df457610df3610d97565b5b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b60006002820490506001821680610e4157607f821691505b602082108103610e5457610e53610dfa565b5b50919050565b6000610e6582610b9d565b9150610e7083610b9d565b9250828203905081811115610e8857610e87610d97565b5b9291505056fea2646970667358221220e2bbe4e79fbff010e16625cff639a72785d4dbf62ac4a60275168b532643766464736f6c63430008180033",
  "storage": [
    {
      "key": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "value": "0x57726170706564204e696269727500000000000000000000000000000000001c"
    },
    {
      "key": "0x0000000000000000000000000000000000000000000000000000000000000001",
      "value": "0x574e49424900000000000000000000000000000000000000000000000000000a"
    },
    {
      "key": "0x0000000000000000000000000000000000000000000000000000000000000002",
      "value": "0x0000000000000000000000000000000000000000000000000000000000000012"
    }
  ]
}
EOF
add_genesis_param_slurpfile '.app_state.evm.accounts += $wnibi_evm' "wnibi_evm" "$WNIBI_EVM_GENESIS_JSON"

# hack for localnet since we don't have a pricefeeder yet
price_btc="50000"
price_eth="2000"
add_genesis_param '.app_state.oracle.exchange_rates[0].pair = "ubtc:uusd"'
add_genesis_param '.app_state.oracle.exchange_rates[0].exchange_rate = "'"$price_btc"'"'
add_genesis_param '.app_state.oracle.exchange_rates[1].pair = "ueth:uusd"'
add_genesis_param '.app_state.oracle.exchange_rates[1].exchange_rate = "'"$price_eth"'"'

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
