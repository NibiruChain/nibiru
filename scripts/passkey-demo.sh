#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="$ROOT/logs"
mkdir -p "$LOG_DIR"

RPC_URL="${JSON_RPC_ENDPOINT:-http://127.0.0.1:8545}"
BUNDLER_PORT="${BUNDLER_PORT:-4337}"
BUNDLER_URL="${BUNDLER_URL:-http://127.0.0.1:${BUNDLER_PORT}}"
BUNDLER_PRIVATE_KEY="${BUNDLER_PRIVATE_KEY:-0x68e80819679abccddfa31064ea84b2fe6870b1eaa0ebe2a1ff40a38533cfab8b}" # guard-cream EVM key
BUNDLER_ADDRESS="${BUNDLER_ADDRESS:-}"
# Amount to send via bank send (micro NIBI)
BUNDLER_FUND_AMOUNT_UNIBI="${BUNDLER_FUND_AMOUNT_UNIBI:-10000000000}" # 10,000 NIBI in micro
BUNDLER_MIN_BALANCE_WEI="${BUNDLER_MIN_BALANCE_WEI:-1000000000000000000}" # 1 NIBI

PASSKEY_CACHE="$ROOT/evm-e2e/.cache/passkey-demo.json"
LOCALNET_PID_FILE="$LOG_DIR/passkey-localnet.pid"
BUNDLER_PID_FILE="$LOG_DIR/passkey-bundler.pid"
BUNDLER_LOG="$LOG_DIR/passkey-bundler.log"

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required for passkey-demo (to read deployment outputs). Please install jq and retry."
  exit 1
fi

if [ -z "$BUNDLER_ADDRESS" ]; then
  # Known keys shortcut to avoid needing ethers when node_modules is missing
  if [ "$BUNDLER_PRIVATE_KEY" = "0x68e80819679abccddfa31064ea84b2fe6870b1eaa0ebe2a1ff40a38533cfab8b" ]; then
    BUNDLER_ADDRESS="0xC0f4b45712670cf7865A14816bE9Af9091EDdA1d"
  elif [ "$BUNDLER_PRIVATE_KEY" = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" ]; then
    BUNDLER_ADDRESS="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
  fi
fi

if [ -z "$BUNDLER_ADDRESS" ]; then
  # Try to derive from the private key using ethers in evm-e2e/node_modules
  BUNDLER_ADDRESS="$(NODE_PATH="$ROOT/evm-e2e/node_modules" node -e "try { const { Wallet } = require('ethers'); console.log(new Wallet(process.env.PK).address) } catch (e) { process.exit(1) }" PK="$BUNDLER_PRIVATE_KEY" 2>/dev/null || true)"
fi

if [ -z "$BUNDLER_ADDRESS" ]; then
  echo "Failed to derive bundler address from BUNDLER_PRIVATE_KEY. Set BUNDLER_ADDRESS explicitly or install ethers in evm-e2e/node_modules."
  exit 1
fi

check_rpc() {
  curl -s -o /dev/null --fail --max-time 2 -X POST "$RPC_URL" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
}

wait_for_rpc() {
  local attempts=0
  until check_rpc; do
    attempts=$((attempts + 1))
    if [ "$attempts" -gt 60 ]; then
      echo "RPC not responding after 60s on $RPC_URL"
      exit 1
    fi
    sleep 1
  done
}

check_bundler() {
  curl -s -o /dev/null --fail --max-time 2 -X POST "$BUNDLER_URL" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","method":"eth_supportedEntryPoints","params":[],"id":1}'
}

wait_for_bundler() {
  local attempts=0
  until check_bundler; do
    attempts=$((attempts + 1))
    if [ "$attempts" -gt 60 ]; then
      echo "Bundler not responding after 60s on $BUNDLER_URL"
      exit 1
    fi
    sleep 1
  done
}

cleanup_bundler() {
  if [ -f "$BUNDLER_PID_FILE" ]; then
    old_pid=$(cat "$BUNDLER_PID_FILE" || true)
    if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
      kill "$old_pid" || true
      sleep 1
    fi
    rm -f "$BUNDLER_PID_FILE"
  fi
  if command -v lsof >/dev/null 2>&1; then
    port_pids=$(lsof -ti :"$BUNDLER_PORT" || true)
    if [ -n "$port_pids" ]; then
      echo "Killing existing process on port $BUNDLER_PORT: $port_pids"
      kill $port_pids || true
      sleep 1
    fi
  fi
}

get_eth_balance() {
  local addr="$1"
  curl -s --fail --max-time 2 -X POST "$RPC_URL" \
    -H 'Content-Type: application/json' \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$addr\",\"latest\"],\"id\":1}" \
    | jq -r '.result // "0x0"' || echo "0x0"
}

ensure_bundler_funded() {
  local balance_hex
  balance_hex="$(get_eth_balance "$BUNDLER_ADDRESS")"
  local balance_dec
  balance_dec="$(node -e "console.log(BigInt('$balance_hex').toString())" 2>/dev/null || echo "0")"

  if [ "$balance_dec" -ge "$BUNDLER_MIN_BALANCE_WEI" ]; then
    echo "Bundler balance is sufficient (${balance_dec} wei) for $BUNDLER_ADDRESS"
    return
  fi

  local bundler_bech32
  bundler_bech32="$(nibid debug addr "${BUNDLER_ADDRESS#0x}" 2>/dev/null | awk '/Bech32 Acc:/ {print $3}')"
  if [ -z "$bundler_bech32" ]; then
    echo "Failed to derive bech32 address for bundler ($BUNDLER_ADDRESS); skipping auto-fund."
    return
  fi

  echo "Funding bundler ($BUNDLER_ADDRESS / $bundler_bech32) from guard-cream validator (amount: ${BUNDLER_FUND_AMOUNT_UNIBI}unibi)..."
  nibid tx bank send validator "$bundler_bech32" "${BUNDLER_FUND_AMOUNT_UNIBI}unibi" \
    --from validator --yes --keyring-backend test --gas auto --gas-adjustment 1.4 \
    --gas-prices 0.25unibi --node http://localhost:26657 --chain-id nibiru-localnet-0 >/dev/null

  echo "Waiting for bundler balance to update..."
  sleep 3
  balance_hex="$(get_eth_balance "$BUNDLER_ADDRESS")"
  balance_dec="$(node -e "console.log(BigInt('$balance_hex').toString())" 2>/dev/null || echo "0")"
  echo "Bundler balance now ${balance_dec} wei"
}

if ! check_rpc; then
  echo "Starting Nibiru localnet (make localnet --no-build)..."
  make localnet FLAGS="--no-build" >"$LOG_DIR/passkey-localnet.log" 2>&1 &
  echo $! >"$LOCALNET_PID_FILE"
  echo "Waiting for RPC at $RPC_URL..."
  wait_for_rpc
else
  echo "RPC already responding at $RPC_URL"
fi

echo "Compiling contracts (hardhat)..."
(cd "$ROOT/evm-e2e" && npx hardhat compile --show-stack-traces >/dev/null)

echo "Deploying EntryPoint + PasskeyAccountFactory and writing passkey-app/.env.local"
(cd "$ROOT/evm-e2e" && node scripts/passkey-demo-setup.js)

if [ ! -f "$PASSKEY_CACHE" ]; then
  echo "Could not find $PASSKEY_CACHE; skipping bundler start."
  echo "Done. Start the UI with: cd passkey-app && npm run dev"
  exit 0
fi

ENTRY_POINT="$(jq -r .entryPoint "$PASSKEY_CACHE")"
CHAIN_ID="$(jq -r .chainId "$PASSKEY_CACHE")"
FACTORY_ADDR="$(jq -r .passkeyFactory "$PASSKEY_CACHE")"

echo "Restarting passkey bundler on $BUNDLER_URL (logs: $BUNDLER_LOG)"
cleanup_bundler
(
  cd "$ROOT/evm-e2e/passkey-sdk"
  if [ ! -d node_modules ]; then
    echo "Installing passkey-sdk dependencies..."
    npm install >/dev/null
  fi
  ENTRY_POINT="$ENTRY_POINT" \
  FACTORY_ADDR="$FACTORY_ADDR" \
  JSON_RPC_ENDPOINT="$RPC_URL" \
  CHAIN_ID="$CHAIN_ID" \
  BUNDLER_PORT="$BUNDLER_PORT" \
  BUNDLER_PRIVATE_KEY="$BUNDLER_PRIVATE_KEY" \
    npm run bundler:local >>"$BUNDLER_LOG" 2>&1 &
  echo $! >"$BUNDLER_PID_FILE"
)
echo "Waiting for bundler at $BUNDLER_URL..."
wait_for_bundler

ensure_bundler_funded

echo "Done. Start the UI with: cd passkey-app && npm run dev"
