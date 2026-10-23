#!/usr/bin/env bash
# Compatibility entrypoint for the maintained CosmWasm optimizer.
set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/../.." && pwd)"
cd -- "$repo_root"
exec cargo run --locked --manifest-path "$repo_root/Cargo.toml" \
  --package sai-wasm-optimizer -- build --manifest-path "$repo_root/Cargo.toml" \
  --output-dir "$repo_root/wasm-contracts/artifacts" --mode "${1:-dev}"
