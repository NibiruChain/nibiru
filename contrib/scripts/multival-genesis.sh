#!/usr/bin/env bash
#
# multival-genesis.sh: Create one genesis for N Nibiru validators on one
# chain, with a gentx from each validator. Runs once in the
# `multival-genesis` service of the Chaosnet `multival` profile (see
# spec/02-chaosnet.md) and writes everything under $NET_DIR:
#
#   $NET_DIR/val-<i>/   node home for validator i (config, data, keyring)
#   $NET_DIR/load/      keyring with the funded bank and Wasm load accounts
#   $NET_DIR/DONE       marker; the script exits early when it exists
#
# Env:
#   MULTIVAL_N        number of validators (default 4)
#   CHAIN_ID          chain ID (default nibiru-localnet-3)
#   NET_DIR           output directory (default /net)
#   EVM_LOAD_ADDRS    space-separated 0x addresses to fund for EVM load

set -euo pipefail

MULTIVAL_N="${MULTIVAL_N:-4}"
CHAIN_ID="${CHAIN_ID:-nibiru-localnet-3}"
NET_DIR="${NET_DIR:-/net}"
EVM_LOAD_ADDRS="${EVM_LOAD_ADDRS:-}"

FUNDS="100000000000000unibi"
SELF_BOND="900000000unibi"

# log: Print a progress line to stderr.
log() {
  echo "[multival-genesis] $*" >&2
}

# val_home: Print the node home of validator $1.
val_home() {
  echo "$NET_DIR/val-$1"
}

# set_genesis: Apply jq filter $1 to genesis file $2 in place.
set_genesis() {
  local filter="$1" file="$2"
  jq "$filter" "$file" > "$file.tmp"
  mv "$file.tmp" "$file"
}

# configure_node: Edit config.toml and app.toml of validator $1 so it binds
# on all interfaces, peers with the other validators, and serves JSON-RPC.
configure_node() {
  local i="$1" peers="$2"
  local home config app
  home="$(val_home "$i")"
  config="$home/config/config.toml"
  app="$home/config/app.toml"

  sed -i 's|^laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|' "$config"
  sed -i "s|^persistent_peers = .*|persistent_peers = \"$peers\"|" "$config"
  sed -i 's|^addr_book_strict = .*|addr_book_strict = false|' "$config"
  sed -i 's|^allow_duplicate_ip = .*|allow_duplicate_ip = true|' "$config"

  # [json-rpc] is the EVM JSON-RPC server.
  sed -i '/^\[json-rpc\]/,/^\[/ s|^enable = false|enable = true|' "$app"
  sed -i 's|^address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|' "$app"
  sed -i 's|^ws-address = "127.0.0.1:8546"|ws-address = "0.0.0.0:8546"|' "$app"
}

main() {
  if [[ -f "$NET_DIR/DONE" ]]; then
    log "genesis already exists in $NET_DIR, keeping it"
    exit 0
  fi
  rm -rf "${NET_DIR:?}"/*
  mkdir -p "$NET_DIR"

  local i home node_ids=() val_addrs=()
  for ((i = 0; i < MULTIVAL_N; i++)); do
    home="$(val_home "$i")"
    nibid init "val-$i" --chain-id "$CHAIN_ID" --home "$home" >/dev/null 2>&1
    nibid keys add validator --keyring-backend test --home "$home" \
      --output json >/dev/null 2>&1
    val_addrs+=("$(nibid keys show validator -a --keyring-backend test --home "$home")")
    node_ids+=("$(nibid tendermint show-node-id --home "$home")")
  done

  # Load accounts: one bank sender and one Wasm sender per validator.
  local load_home="$NET_DIR/load" load_addrs=()
  for ((i = 0; i < MULTIVAL_N; i++)); do
    for kind in bank wasm; do
      nibid keys add "$kind-$i" --keyring-backend test --home "$load_home" \
        --output json >/dev/null 2>&1
      load_addrs+=("$(nibid keys show "$kind-$i" -a --keyring-backend test --home "$load_home")")
    done
  done

  local evm_addr
  for evm_addr in $EVM_LOAD_ADDRS; do
    load_addrs+=("$(nibid debug addr "${evm_addr#0x}" | awk '/^Bech32 Acc:/ {print $3}')")
  done

  # Validator 0 builds the genesis. Every account goes in before any gentx.
  local home0 genesis0 addr
  home0="$(val_home 0)"
  genesis0="$home0/config/genesis.json"
  for addr in "${val_addrs[@]}" "${load_addrs[@]}"; do
    nibid genesis add-genesis-account "$addr" "$FUNDS" --home "$home0"
  done
  set_genesis ".app_state.sudo.sudoers.root = \"${val_addrs[0]}\"" "$genesis0"

  mkdir -p "$home0/config/gentx"
  for ((i = 0; i < MULTIVAL_N; i++)); do
    home="$(val_home "$i")"
    if [[ "$i" -ne 0 ]]; then
      cp "$genesis0" "$home/config/genesis.json"
    fi
    nibid genesis gentx validator "$SELF_BOND" --chain-id "$CHAIN_ID" \
      --moniker "val-$i" --keyring-backend test --home "$home" \
      --output-document "$home0/config/gentx/gentx-val-$i.json"
  done
  nibid genesis collect-gentxs --home "$home0" >/dev/null 2>&1
  nibid genesis validate-genesis --home "$home0"

  local peers j
  for ((i = 0; i < MULTIVAL_N; i++)); do
    peers=""
    for ((j = 0; j < MULTIVAL_N; j++)); do
      [[ "$j" -eq "$i" ]] && continue
      peers+="${peers:+,}${node_ids[$j]}@nibiru-val-$j:26656"
    done
    if [[ "$i" -ne 0 ]]; then
      cp "$genesis0" "$(val_home "$i")/config/genesis.json"
    fi
    configure_node "$i" "$peers"
  done

  touch "$NET_DIR/DONE"
  log "created $MULTIVAL_N validators on $CHAIN_ID in $NET_DIR"
}

main "$@"
