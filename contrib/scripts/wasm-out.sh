#!/usr/bin/env bash
#
# wasm-out.sh:
# Compile contracts into .wasm bytecode optimizer to reduce gas and
# have deterministic behavior in wasmer.
#
# Ref: https://github.com/CosmWasm/optimizer

set -e
script_path=$(dirname "$(readlink -f "$0")")
# shellcheck source=./bashlib.sh
source "$script_path/bashlib.sh"

# IMAGE_VERSION="0.15.0" # Use v0.14 for Wasm VM v1 (Wasmd v0.44)
# IMAGE="cosmwasm/workspace-optimizer" # preserved but deprecated v0.15+
# On Docker Hub: https://hub.docker.com/r/cosmwasm/workspace-optimizer

# For later use when we migrate to Wasm VM v2.
IMAGE_VERSION="0.17.0"
IMAGE="cosmwasm/optimizer"
# On Docker Hub: https://hub.docker.com/r/cosmwasm/optimizer

# Compiles CosmWasm smart contracts to WebAssembly bytecode (.wasm)
wasm() {
  docker run --rm -v "$(pwd)":/code \
    --mount type=volume,source="$(basename "$(pwd)")_cache",target=/code/target \
    --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
    "$IMAGE:$IMAGE_VERSION"
}

main() {
  echo "Running Wasm bytecode optimizer for all contracts"
  echo "IMAGE: $IMAGE"
  echo "IMAGE_VERSION: $IMAGE_VERSION"
  if wasm; then
    echo "🔥 Compiled all smart contracts successfully. "
    return 0
  fi
  return 1
}

if ! main "$@"; then
  log_error "Compilation failed."
fi