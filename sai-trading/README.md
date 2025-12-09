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

# Optional: Slack notifications
SLACK_WEBHOOK=""
SLACK_ERROR_KEYWORDS=""
```

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

### Example `.env` file

```bash
# Account
EVM_MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
SLACK_WEBHOOK=""
SLACK_ERROR_KEYWORDS=""
```

**Note**: The `.env` file is automatically loaded if present. You can also pass values via command-line flags, which take precedence over environment variables.

