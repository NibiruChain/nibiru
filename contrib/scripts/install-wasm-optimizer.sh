#!/usr/bin/env bash
# Install a pinned native optimizer release through cargo-binstall.
set -euo pipefail

version_pattern='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-rc\.[1-9][0-9]*)?$'
if [[ $# -lt 1 || $# -gt 2 || ! "$1" =~ $version_pattern ]]; then
  echo "Usage: $0 VERSION [INSTALL_ROOT]" >&2
  exit 2
fi
version="$1"
for program in curl cargo-binstall awk; do
  if ! command -v "$program" >/dev/null 2>&1; then
    echo "$program is required. See lib/cosmwasm-optimizer/README.md." >&2
    exit 127
  fi
done
if command -v sha256sum >/dev/null 2>&1; then
  hash_command=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
  hash_command=(shasum -a 256)
else
  echo "sha256sum or shasum is required." >&2
  exit 127
fi

temporary="$(mktemp -d)"
trap 'rm -rf -- "$temporary"' EXIT
base="https://github.com/NibiruChain/nibiru/releases/download/sai-wasm-optimizer/v${version}"
manifest="$temporary/sai-wasm-optimizer.toml"
curl --proto '=https' --tlsv1.2 --fail --location --silent --show-error \
  "$base/sai-wasm-optimizer.toml" --output "$manifest"
curl --proto '=https' --tlsv1.2 --fail --location --silent --show-error \
  "$base/SHA256SUMS" --output "$temporary/SHA256SUMS"
expected="$(awk '$2 == "sai-wasm-optimizer.toml" { print $1 }' "$temporary/SHA256SUMS")"
actual="$("${hash_command[@]}" "$manifest" | awk '{ print $1 }')"
if [[ ! "$expected" =~ ^[a-f0-9]{64}$ || "$actual" != "$expected" ]]; then
  echo "Optimizer release manifest checksum mismatch." >&2
  exit 1
fi
# Binstall accepts a requested version override without checking the manifest's
# version. Validate the pinned manifest and let Binstall use that version alone.
manifest_version="$(awk '
  /^\[package\]$/ { package = 1; next }
  /^\[/ { package = 0 }
  package && /^version = / { gsub(/"/, "", $3); print $3 }
' "$manifest")"
if [[ "$manifest_version" != "$version" ]]; then
  echo "Optimizer manifest version does not match requested release $version." >&2
  exit 1
fi

install_args=(--manifest-path "$manifest" --strategies crate-meta-data --no-confirm)
if [[ $# -eq 2 ]]; then
  install_args+=(--root "$2")
fi
cargo-binstall "${install_args[@]}" sai-wasm-optimizer
