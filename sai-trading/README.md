# Nibiru/Sai-Trading

Rough notes (for now)

## Plan

- [ ] Use sai-perps version for both the EVM interface and Wasm contracts.

---

### Pulling in artifacts

Sketching out the flow from the "sai-perps" repo:

```bash
root="$(pwd)" # Nibiru/sai-trading

# Assuming sai-perps is temporarily locally cloned:
# npx degit ... OR downlaod from release assets
sai_perps="$root/sai-perps" 

cp "$sai_perps/artifacts/*" artifacts/
(cd $sai_perps just evm-install && just evm-build)
rm -rf artifacts/solidity
cp -r "$sai_perps/evm-interface/artifacts" artifacts/solidity
```

cp "

```bash
cp "$sai_perps/evm-interface/artifacts/contracts/PerpVaultEvmInterface.sol/PerpVaultEvmInterface.json" artifacts/
jq '{sourceName, contractName, abi, bytecode}' artifacts/PerpVaultEvmInterface.json > tmp.json
mv tmp.json artifacts/PerpVaultEvmInterface.json
```

### yq for artifacts build info:

The `yq` tool is written in Go as a dependency free binary.

```bash
go install github.com/mikefarah/yq/v4@latest
```
https://github.com/mikefarah/yq?tab=readme-ov-file#github-action

---

## Running the EVM Trader

### Configuration

Create a `.env` file in the root directory (see `.env.example`):

```bash
# Account credentials (use either private key OR mnemonic)
EVM_PRIVATE_KEY=0x1234567890abcdef...
# OR
EVM_MNEMONIC="word1 word2 word3 ..."

SLACK_WEBHOOK=""
```

#### Slack Notification Configuration (Optional)

Slack notifications are optional. If not configured, errors will still be logged to stdout.

**Step 1: Set Slack Webhook**

Set your Slack webhook URL in `.env` file:

```bash
SLACK_WEBHOOK="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

**Step 2: Configure Error Filters**

Configure which errors to send to Slack in `networks.toml`:

```toml
[notifications.filters]
include = ["insufficient funds", "gas error", "execution failed"]
exclude = ["expected error", "test"]
```

**Filter Logic:**
- **Exclude list**: If any keyword matches, skip the notification (highest priority)
- **Include list**: If not empty, only send if at least one keyword matches
- **Empty filters**: Send all errors to Slack

### Running the trader

**Auto trading** :
```bash
just run-trader auto --network testnet
```

With custom parameters:
```bash
just run-trader auto --market-index 0 --min-leverage 5 --max-leverage 20 --blocks-before-close 20 --network testnet
```

Or use a config file:
```bash
go run ./cmd/trader auto --config auto-trader.localnet.json
```

**Manual trading**:
```bash
# Open a single position
just run-trader open --trade-type trade --market-index 0 --long false --trade-size 1 --network testnet

# Using JSON file
just run-trader --trade-json sample_txs/open_trade.json
```

### Available flags

**Root flags (shared across all commands):**
- `--network`: Network mode (`localnet`, `testnet`, `mainnet`) (default: `localnet`)
- `--evm-rpc`: EVM RPC URL (overrides network mode default)
- `--networks-toml`: Path to networks TOML configuration file (default: `networks.toml`)
- `--contracts-env`: Path to contracts env file (legacy, overrides networks.toml)
- `--private-key`: Private key in hex format (overrides `EVM_PRIVATE_KEY` env var)
- `--mnemonic`: BIP39 mnemonic phrase (overrides `EVM_MNEMONIC` env var)

**Auto command flags:**
- `--config`: Path to JSON config file (optional)
- `--market-index`: Market index to trade (default: 0)
- `--collateral-index`: Collateral token index (default: 0, uses market's quote token)
- `--min-trade-size`: Minimum trade size in smallest units (default: 1000000)
- `--max-trade-size`: Maximum trade size in smallest units (default: 5000000)
- `--min-leverage`: Minimum leverage (default: 1, e.g., 1 for 1x)
- `--max-leverage`: Maximum leverage (default: 10, e.g., 10 for 10x)
- `--blocks-before-close`: Number of blocks to wait before closing a position (default: 20)
- `--max-open-positions`: Maximum number of positions to keep open at once (default: 5)
- `--loop-delay`: Delay in seconds between each loop iteration (default: 5)
