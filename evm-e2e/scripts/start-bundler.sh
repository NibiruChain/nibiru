#!/usr/bin/env bash
set -euo pipefail

# Simple helper to launch a Stackup-style ERC-4337 bundler in Docker.
# Env vars:
#   RPC_URL        (required) - Nibiru EVM RPC endpoint (http://127.0.0.1:8545)
#   ENTRY_POINT    (required) - Deployed EntryPoint address
#   CHAIN_ID       (required) - Nibiru EVM chain id
#   PRIVATE_KEY    (optional) - Bundler signer key; defaults to a dev key
#   BUNDLER_PORT   (optional) - Host port to expose (default 4337)
#   BUNDLER_IMAGE  (optional) - Docker image (default ghcr.io/stackup-wallet/stackup-bundler:latest)
#
# Usage:
#   RPC_URL=... ENTRY_POINT=... CHAIN_ID=... ./scripts/start-bundler.sh

if [[ -z "${RPC_URL:-}" || -z "${ENTRY_POINT:-}" || -z "${CHAIN_ID:-}" ]]; then
  echo "RPC_URL, ENTRY_POINT, and CHAIN_ID are required env vars."
  exit 1
fi

PRIVATE_KEY="${PRIVATE_KEY:-0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d}" # dev key
# strip 0x for images that expect raw hex
PRIVATE_KEY="${PRIVATE_KEY#0x}"
BUNDLER_PORT="${BUNDLER_PORT:-4337}"
BUNDLER_IMAGE="${BUNDLER_IMAGE:-ghcr.io/stackup-wallet/stackup-bundler:latest}"

CONFIG_JSON="$(mktemp)"
cat > "${CONFIG_JSON}" <<EOF
{
  "network": {
    "rpcUrl": "${RPC_URL}",
    "chainId": ${CHAIN_ID}
  },
  "entryPoints": ["${ENTRY_POINT}"],
  "bundler": {
    "port": ${BUNDLER_PORT}
  },
  "wallet": {
    "privateKey": "${PRIVATE_KEY}"
  }
}
EOF

echo "Bundler config written to ${CONFIG_JSON}"
echo "Using image: ${BUNDLER_IMAGE}"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker not found. Install Docker or run bundler manually with the config above."
  exit 1
fi

# Stackup OSS image expects explicit env vars rather than a config file.
if [[ "${BUNDLER_IMAGE}" == *"stackup-bundler"* ]]; then
  docker run --rm \
    -p "${BUNDLER_PORT}:${BUNDLER_PORT}" \
    -e ERC4337_BUNDLER_ETH_CLIENT_URL="${RPC_URL}" \
    -e ERC4337_BUNDLER_ENTRY_POINT="${ENTRY_POINT}" \
    -e ERC4337_BUNDLER_PRIVATE_KEY="${PRIVATE_KEY}" \
    -e ERC4337_BUNDLER_PORT="${BUNDLER_PORT}" \
    "${BUNDLER_IMAGE}"
else
  docker run --rm \
    -p "${BUNDLER_PORT}:${BUNDLER_PORT}" \
    -v "${CONFIG_JSON}":/app/config.json \
    -e BUNDLER_CONFIG=/app/config.json \
    "${BUNDLER_IMAGE}"
fi
