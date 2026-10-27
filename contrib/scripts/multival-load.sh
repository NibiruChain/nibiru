#!/usr/bin/env bash
#
# multival-load.sh: Send bank and Wasm transactions to every validator of
# the Chaosnet `multival` profile, so different proposers build blocks from
# different mempools. Started by contrib/scripts/determinism.sh in a one-off
# container on the Compose network. EVM load comes from
# contrib/scripts/determinism (evm-load) over JSON-RPC.
#
# Uses the bank-<i> and wasm-<i> keys that multival-genesis.sh put in
# $NET_DIR/load. Account i sends through node i. Failed broadcasts are
# counted, not fatal, since validators restart while this runs.
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

# wait_blocks: Wait until node 0's height grows by $1.
wait_blocks() {
  local want start now
  start="$(height)"
  want=$((start + $1))
  for _ in $(seq 1 60); do
    now="$(height)"
    [[ "$now" -ge "$want" ]] && return 0
    sleep 1
  done
  return 1
}

height() {
  nibid status --node "$(node_url 0)" 2>/dev/null \
    | jq -r '.sync_info.latest_block_height // .SyncInfo.latest_block_height // 0' \
    || echo 0
}

# setup_wasm: Store the counter contract and instantiate one copy per
# validator. Prints the contract addresses, one per line.
setup_wasm() {
  local code_id i
  log "storing $WASM_FILE"
  tx 0 wasm store "$WASM_FILE" --from wasm-0 --gas 10000000 >/dev/null
  wait_blocks 2 || return 1
  code_id="$(nibid q wasm list-code --node "$(node_url 0)" --output json \
    | jq -r '.code_infos[-1].code_id')"
  [[ -n "$code_id" && "$code_id" != null ]] || return 1

  for ((i = 0; i < MULTIVAL_N; i++)); do
    tx "$i" wasm instantiate "$code_id" '{"count":0}' --label "counter-$i" \
      --no-admin --from "wasm-$i" --gas 1000000 >/dev/null
  done
  wait_blocks 2 || return 1
  nibid q wasm list-contract-by-code "$code_id" --node "$(node_url 0)" \
    --output json | jq -r '.contracts[]'
}

# send_loop: Until $deadline, send a bank transfer and a Wasm increment from
# account $1 through node $1. Prints "<accepted> <failed>" when done.
send_loop() {
  local i="$1" deadline="$2" to="$3" contract="$4"
  local round=0 ok=0 failed=0 code
  while [[ "$SECONDS" -lt "$deadline" ]]; do
    round=$((round + 1))
    code="$(tx "$i" bank send "bank-$i" "$to" "${round}unibi" --from "bank-$i")"
    if [[ "$code" == 0 ]]; then ok=$((ok + 1)); else failed=$((failed + 1)); fi

    if [[ -n "$contract" ]]; then
      code="$(tx "$i" wasm execute "$contract" '{"increment":{}}' \
        --from "wasm-$i" --gas 500000)"
      if [[ "$code" == 0 ]]; then ok=$((ok + 1)); else failed=$((failed + 1)); fi
    fi
    # Let the last tx land before the account's next sequence number.
    sleep 1
  done
  echo "$ok $failed"
}

main() {
  local contracts=() bank_addrs=() i deadline contract
  mapfile -t contracts < <(setup_wasm || true)
  if [[ "${#contracts[@]}" -eq 0 ]]; then
    log "Wasm setup failed; sending bank load only"
  else
    log "instantiated ${#contracts[@]} counter contracts"
  fi

  for ((i = 0; i < MULTIVAL_N; i++)); do
    bank_addrs+=("$(nibid keys show "bank-$i" -a --keyring-backend test \
      --home "$LOAD_HOME")")
  done

  local results_dir
  results_dir="$(mktemp -d)"
  deadline=$((SECONDS + LOAD_SECONDS))
  for ((i = 0; i < MULTIVAL_N; i++)); do
    contract=""
    if [[ "${#contracts[@]}" -gt 0 ]]; then
      contract="${contracts[$((i % ${#contracts[@]}))]}"
    fi
    send_loop "$i" "$deadline" "${bank_addrs[$(((i + 1) % MULTIVAL_N))]}" \
      "$contract" > "$results_dir/$i" &
  done
  wait

  local ok=0 failed=0 a b
  for ((i = 0; i < MULTIVAL_N; i++)); do
    read -r a b < "$results_dir/$i"
    ok=$((ok + a))
    failed=$((failed + b))
  done
  log "bank and Wasm load: $ok accepted, $failed rejected or failed"
}

main "$@"
