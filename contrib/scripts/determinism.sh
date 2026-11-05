#!/usr/bin/env bash
#
# determinism.sh: Check that 4 Nibiru validators on one chain compute the
# same app hash at every height under EVM, Wasm, and bank load while
# validators restart and get killed. Run it as `just chaos determinism`. See
# spec/02-chaosnet.md.
#
# Steps:
#   1. Build the image and start the Chaosnet `multival` profile from a
#      fresh multival-data volume.
#   2. Store and instantiate the Wasm counter contracts
#      (contrib/scripts/multival-load.sh setup) before any validator is
#      taken down.
#   3. Send EVM load (contrib/scripts/determinism evm-load, JSON-RPC) and
#      bank and Wasm load (multival-load.sh load, nibid CLI) to every node.
#   4. Meanwhile, `docker compose kill -s SIGKILL` or `restart` one validator
#      at a time, so nodes replay blocks from disk on startup.
#   5. Wait for every node to reach a target height and compare /block and
#      /block_results for every height (determinism compare).
#   6. Print the last 100 log lines of any node that stopped, then remove
#      the network.
#
# The check fails when nodes disagree or a node stops. If they agree, it
# still fails unless the load ran: the network reached height 3, Wasm setup
# succeeded, every node accepted EVM transactions, the bank and Wasm load
# had acceptances, and at least MIN_OK_TXS transactions succeeded.
#
# Env:
#   LOAD_SECONDS        load and perturbation time (default 120)
#   EXTRA_HEIGHTS       heights past the end of load to wait for (default 10)
#   MIN_OK_TXS          successful (code 0) txs required (default LOAD_SECONDS)
#   PERTURB             "false" keeps every validator up (default true)
#   STALL_SECONDS       a node whose height does not move this long has
#                       stopped (default 60)
#   MULTIVAL_IMAGE      image tag (default nibiru-multival:local)
#   MULTIVAL_BUILD_TAGS extra Go build tags for nibid (default none)
#   MULTIVAL_IMAGE_VAL2 an already built image for nibiru-val-2 only (default
#                       MULTIVAL_IMAGE)
#   KEEP_NETWORK        "true" leaves the network running (default false)
#
# Exits non-zero when nodes disagree, a node stops, or the load did not run.

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd -P)"
COMPOSE_FILE="$REPO_ROOT/contrib/docker-compose/docker-compose-chaosnet.yml"

LOAD_SECONDS="${LOAD_SECONDS:-120}"
EXTRA_HEIGHTS="${EXTRA_HEIGHTS:-10}"
MIN_OK_TXS="${MIN_OK_TXS:-$LOAD_SECONDS}"
PERTURB="${PERTURB:-true}"
STALL_SECONDS="${STALL_SECONDS:-60}"
KEEP_NETWORK="${KEEP_NETWORK:-false}"
export MULTIVAL_IMAGE="${MULTIVAL_IMAGE:-nibiru-multival:local}"
export MULTIVAL_BUILD_TAGS="${MULTIVAL_BUILD_TAGS:-}"

N=4
NAMES="val-0,val-1,val-2,val-3"
RPCS="http://127.0.0.1:46657,http://127.0.0.1:46658,http://127.0.0.1:46659,http://127.0.0.1:46660"
EVM_RPCS="http://127.0.0.1:48545,http://127.0.0.1:48546,http://127.0.0.1:48547,http://127.0.0.1:48548"
WASM_FILE="$REPO_ROOT/evm/precompile/test/hello_world_counter.wasm"

WORK_DIR="$(mktemp -d)"
TOOL="$WORK_DIR/determinism"
BG_PIDS=()
# LOAD_FAILURES collects why the load did not run as required.
LOAD_FAILURES=()

# log: Print a progress line to stderr.
log() {
  echo "[determinism] $*" >&2
}

compose() {
  docker compose -f "$COMPOSE_FILE" --profile multival "$@"
}

services() {
  local i
  for ((i = 0; i < N; i++)); do
    echo "nibiru-val-$i"
  done
}

# rpc_height: Print the latest height of validator $1, or 0.
rpc_height() {
  local port=$((46657 + $1))
  curl -sf --max-time 3 "http://127.0.0.1:$port/status" 2>/dev/null \
    | jq -r '.result.sync_info.latest_block_height' 2>/dev/null || echo 0
}

# max_height: Print the highest height among validators.
max_height() {
  local i h max=0
  for ((i = 0; i < N; i++)); do
    h="$(rpc_height "$i")"
    [[ "$h" =~ ^[0-9]+$ ]] || h=0
    ((h > max)) && max="$h"
  done
  echo "$max"
}

# wait_for_height: Wait up to $2 seconds for any validator to reach $1.
wait_for_height() {
  local want="$1" limit="$2" t
  for ((t = 0; t < limit; t++)); do
    [[ "$(max_height)" -ge "$want" ]] && return 0
    sleep 1
  done
  return 1
}

cleanup() {
  local pid
  for pid in "${BG_PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  if [[ "$KEEP_NETWORK" != true ]]; then
    log "removing the multival network"
    compose down --volumes --timeout 1 >/dev/null 2>&1 || true
  fi
  rm -rf "$WORK_DIR"
}

# perturb: Until $1 (a $SECONDS deadline), take down one validator at a
# time, alternating `kill -s SIGKILL` + start and `restart`.
perturb() {
  local deadline="$1" i=0 round=0 svc
  while [[ "$SECONDS" -lt "$deadline" ]]; do
    sleep 10
    [[ "$SECONDS" -lt "$deadline" ]] || break
    svc="nibiru-val-$i"
    if ((round % 2 == 0)); then
      log "perturb: kill -9 $svc"
      compose kill -s SIGKILL "$svc" >/dev/null 2>&1 || true
      sleep 5
      compose start "$svc" >/dev/null 2>&1 || true
    else
      log "perturb: restart $svc"
      compose restart --timeout 5 "$svc" >/dev/null 2>&1 || true
    fi
    round=$((round + 1))
    i=$(((i + 1) % N))
  done
}

# multival_load: Run contrib/scripts/multival-load.sh $1 in a one-off
# container on the Compose network.
multival_load() {
  compose run --rm --no-deps -T \
    -v "$WASM_FILE:/wasm/counter.wasm:ro" \
    -e LOAD_SECONDS="$LOAD_SECONDS" \
    --entrypoint bash multival-genesis /scripts/multival-load.sh "$1"
}

# send_load: Send EVM, bank, and Wasm load for LOAD_SECONDS while perturbing
# validators, and record every load job that failed in LOAD_FAILURES.
send_load() {
  local deadline=$((SECONDS + LOAD_SECONDS)) evm_pid cli_pid

  "$TOOL" evm-load -rpc "$EVM_RPCS" -duration "${LOAD_SECONDS}s" &
  evm_pid="$!"
  multival_load load &
  cli_pid="$!"
  BG_PIDS=("$evm_pid" "$cli_pid")

  if [[ "$PERTURB" == true ]]; then
    perturb "$deadline"
  fi
  wait "$evm_pid" || LOAD_FAILURES+=("EVM load failed")
  wait "$cli_pid" || LOAD_FAILURES+=("bank and Wasm load failed")
  BG_PIDS=()
}

main() {
  trap cleanup EXIT

  log "building the determinism tool"
  (cd "$REPO_ROOT" && go build -o "$TOOL" ./contrib/scripts/determinism)

  log "building $MULTIVAL_IMAGE (extra tags: ${MULTIVAL_BUILD_TAGS:-none})"
  compose build multival-genesis

  log "starting $N validators from a fresh genesis"
  compose down --volumes --timeout 1 >/dev/null 2>&1 || true
  # shellcheck disable=SC2046
  compose up --detach multival-genesis $(services)

  local start=$SECONDS
  if wait_for_height 3 90; then
    log "network is producing blocks; setting up Wasm contracts"
    if multival_load setup; then
      log "sending load for ${LOAD_SECONDS}s"
      send_load
    else
      LOAD_FAILURES+=("Wasm setup failed")
    fi
  else
    log "network did not reach height 3; skipping load"
    LOAD_FAILURES+=("network did not reach height 3")
  fi

  local target
  target=$(($(max_height) + EXTRA_HEIGHTS))
  log "comparing every node up to height $target"
  local failed_file="$WORK_DIR/failed" status=0
  "$TOOL" compare -rpc "$RPCS" -names "$NAMES" -target "$target" \
    -stall "${STALL_SECONDS}s" -min-ok-txs "$MIN_OK_TXS" \
    -failed-out "$failed_file" || status=$?

  if [[ -s "$failed_file" ]]; then
    local name
    log "stopped: $(paste -sd, "$failed_file")"
    while IFS= read -r name; do
      log "last 100 log lines of nibiru-$name:"
      compose logs --no-color --tail 100 "nibiru-$name" || true
      compose logs --no-color "nibiru-$name" 2>/dev/null \
        | grep -m 3 "wrong Block.Header.AppHash" || true
    done < "$failed_file"
  fi

  log "run took $((SECONDS - start))s after startup"
  if [[ "$status" -ne 0 ]]; then
    log "FAIL: nodes disagree, stopped, or ran too few txs (see above)"
    exit "$status"
  fi
  if [[ "${#LOAD_FAILURES[@]}" -gt 0 ]]; then
    local reasons
    reasons="$(printf '%s; ' "${LOAD_FAILURES[@]}")"
    log "FAIL: load did not run as required: ${reasons%; }"
    exit 1
  fi
  log "OK: all $N validators agree at every height up to $target"
}

main "$@"
