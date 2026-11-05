#!/usr/bin/env bash
#
# multival-load.sh: Send bank and Wasm transactions to every validator of
# the Chaosnet `multival` profile, so different proposers build blocks from
# different mempools. Started by contrib/scripts/determinism.sh in a one-off
# container on the Compose network. EVM load comes from
# contrib/scripts/determinism (evm-load) over JSON-RPC.
#
# Uses the bank-<i> and wasm-<i> keys that multival-genesis.sh put in
# $NET_DIR/load. Account i sends through node i.
#
# Subcommands:
#   setup  Store the counter contract and instantiate one copy per
#          validator, retrying through each node in turn. Writes the
#          addresses to $NET_DIR/load/contracts. Exits 1 if setup fails.
#   load   Until LOAD_SECONDS pass, send bank transfers and Wasm increments.
#          Single failed broadcasts are counted, not fatal, since validators
#          restart while this runs. Exits 1 if no bank or no Wasm
#          transaction was accepted.
#
# Env:
#   MULTIVAL_N      number of validators (default 4)
#   CHAIN_ID        chain ID (default nibiru-localnet-3)
#   NET_DIR         genesis output directory (default /net)
#   LOAD_SECONDS    how long to send (default 60)
#   WASM_FILE       contract to store (default /wasm/counter.wasm), a
#                   counter with {"increment":{}}

set -uo pipefail

MULTIVAL_N="${MULTIVAL_N:-4}"
CHAIN_ID="${CHAIN_ID:-nibiru-localnet-3}"
NET_DIR="${NET_DIR:-/net}"
LOAD_SECONDS="${LOAD_SECONDS:-60}"
WASM_FILE="${WASM_FILE:-/wasm/counter.wasm}"

LOAD_HOME="$NET_DIR/load"
CONTRACTS_FILE="$LOAD_HOME/contracts"

# log: Print a progress line to stderr.
log() {
  echo "[multival-load] $*" >&2
}

# node_url: Print the RPC URL of validator $1 on the Compose network.
node_url() {
  echo "tcp://nibiru-val-$1:26657"
}

# tx: Run `nibid tx` with the load keyring against node $1, and print the
# CheckTx code (or "err" when the broadcast fails).
tx() {
  local node="$1"
  shift
  local out
  out="$(nibid tx "$@" \
    --node "$(node_url "$node")" --chain-id "$CHAIN_ID" \
    --keyring-backend test --home "$LOAD_HOME" \
    --broadcast-mode sync --output json --yes 2>/dev/null)" || {
    echo err
    return
  }
  jq -r '.code // "err"' <<< "$out" 2>/dev/null || echo err
}

# wait_blocks: Wait until node $1's height grows by $2.
wait_blocks() {
  local node="$1" want start now
  start="$(height "$node")"
  want=$((start + $2))
  for _ in $(seq 1 60); do
    now="$(height "$node")"
    [[ "$now" -ge "$want" ]] && return 0
    sleep 1
  done
  return 1
}

# height: Print the latest height of node $1, or 0.
height() {
  nibid status --node "$(node_url "$1")" 2>/dev/null \
    | jq -r '.sync_info.latest_block_height // .SyncInfo.latest_block_height // 0' \
    || echo 0
}

# setup_wasm: Store the counter contract through node $1 and instantiate one
# copy per validator. Prints the contract addresses, one per line, and
# returns 1 unless every copy was instantiated.
setup_wasm() {
  local node="$1" code_id i out
  log "storing $WASM_FILE through nibiru-val-$node"
  tx "$node" wasm store "$WASM_FILE" --from wasm-0 --gas 10000000 >/dev/null
  wait_blocks "$node" 2 || return 1
  code_id="$(nibid q wasm list-code --node "$(node_url "$node")" \
    --output json 2>/dev/null | jq -r '.code_infos[-1].code_id')"
  [[ -n "$code_id" && "$code_id" != null ]] || return 1

  for ((i = 0; i < MULTIVAL_N; i++)); do
    tx "$node" wasm instantiate "$code_id" '{"count":0}' \
      --label "counter-$i" --no-admin --from "wasm-$i" --gas 1000000 >/dev/null
  done
  wait_blocks "$node" 2 || return 1
  out="$(nibid q wasm list-contract-by-code "$code_id" \
    --node "$(node_url "$node")" --output json 2>/dev/null \
    | jq -r '.contracts[]')" || return 1
  [[ "$(grep -c . <<< "$out")" -ge "$MULTIVAL_N" ]] || return 1
  echo "$out"
}

# cmd_setup: Run setup_wasm through nodes 0 to MULTIVAL_N-1 in turn, up to 3
# rounds, and write the addresses to $CONTRACTS_FILE.
cmd_setup() {
  local round node out
  rm -f "$CONTRACTS_FILE"
  for round in 1 2 3; do
    for ((node = 0; node < MULTIVAL_N; node++)); do
      if out="$(setup_wasm "$node")"; then
        echo "$out" > "$CONTRACTS_FILE"
        log "instantiated $(grep -c . <<< "$out") counter contracts"
        return 0
      fi
      log "Wasm setup through nibiru-val-$node failed (round $round)"
    done
  done
  log "FAIL: Wasm setup failed through every node"
  return 1
}

# send_loop: Until $deadline, send a bank transfer and a Wasm increment from
# account $1 through node $1. Prints "<bank ok> <wasm ok> <failed>".
send_loop() {
  local i="$1" deadline="$2" to="$3" contract="$4"
  local round=0 bank_ok=0 wasm_ok=0 failed=0 code
  while [[ "$SECONDS" -lt "$deadline" ]]; do
    round=$((round + 1))
    code="$(tx "$i" bank send "bank-$i" "$to" "${round}unibi" --from "bank-$i")"
    if [[ "$code" == 0 ]]; then
      bank_ok=$((bank_ok + 1))
    else
      failed=$((failed + 1))
    fi

    code="$(tx "$i" wasm execute "$contract" '{"increment":{}}' \
      --from "wasm-$i" --gas 500000)"
    if [[ "$code" == 0 ]]; then
      wasm_ok=$((wasm_ok + 1))
    else
      failed=$((failed + 1))
    fi
    # Let the last tx land before the account's next sequence number.
    sleep 1
  done
  echo "$bank_ok $wasm_ok $failed"
}

# cmd_load: Send bank and Wasm load from every account to its own node.
cmd_load() {
  local contracts=() bank_addrs=() i deadline
  if [[ -s "$CONTRACTS_FILE" ]]; then
    mapfile -t contracts < "$CONTRACTS_FILE"
  fi
  if [[ "${#contracts[@]}" -eq 0 ]]; then
    log "FAIL: no contracts in $CONTRACTS_FILE; run setup first"
    return 1
  fi

  for ((i = 0; i < MULTIVAL_N; i++)); do
    bank_addrs+=("$(nibid keys show "bank-$i" -a --keyring-backend test \
      --home "$LOAD_HOME")")
  done

  local results_dir
  results_dir="$(mktemp -d)"
  deadline=$((SECONDS + LOAD_SECONDS))
  for ((i = 0; i < MULTIVAL_N; i++)); do
    send_loop "$i" "$deadline" "${bank_addrs[$(((i + 1) % MULTIVAL_N))]}" \
      "${contracts[$((i % ${#contracts[@]}))]}" > "$results_dir/$i" &
  done
  wait

  local bank_ok=0 wasm_ok=0 failed=0 a b c
  for ((i = 0; i < MULTIVAL_N; i++)); do
    read -r a b c < "$results_dir/$i"
    bank_ok=$((bank_ok + a))
    wasm_ok=$((wasm_ok + b))
    failed=$((failed + c))
  done
  log "bank and Wasm load: $bank_ok bank and $wasm_ok Wasm accepted," \
    "$failed rejected or failed"
  if [[ "$bank_ok" -eq 0 || "$wasm_ok" -eq 0 ]]; then
    log "FAIL: no bank or no Wasm transaction was accepted"
    return 1
  fi
}

case "${1:-}" in
  setup) cmd_setup ;;
  load) cmd_load ;;
  *)
    echo "usage: multival-load.sh setup|load" >&2
    exit 2
    ;;
esac
