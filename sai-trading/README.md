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

### Configuration via `.env` file

Create a `.env` file in the root directory to configure the trader:

```bash
# Account credentials (use either private key OR mnemonic)
EVM_PRIVATE_KEY=0x1234567890abcdef...  # Your private key in hex format
# OR
EVM_MNEMONIC="word1 word2 word3 ..."   # Your BIP39 mnemonic phrase

### Running the trader

**Dynamic trading** (uses config parameters):
```bash
just run-trader
# or with custom parameters:
just run-trader --market-index 0 --leverage-min 5 --leverage-max 20
```

**Static JSON file trading**:
```bash
just run-trader --trade-json sample_txs/open_trade.json
```

### Available flags

- `--network`: Network mode (`localnet`, `testnet`, `mainnet`)
- `--private-key`: Private key in hex format (overrides `EVM_PRIVATE_KEY` env var)
- `--mnemonic`: BIP39 mnemonic phrase (overrides `EVM_MNEMONIC` env var)
- `--contracts-env`: Path to contracts env file (defaults to `.cache/localnet_contracts.env`)
- `--trade-json`: Path to JSON file with trade parameters (overrides dynamic trading)
- `--market-index`: Market index to trade (default: 0)
- `--collateral-index`: Collateral token index (default: 1)
- `--leverage-min`: Minimum leverage (default: 5)
- `--leverage-max`: Maximum leverage (default: 20)
- `--trade-size-min`: Minimum trade size in smallest units (default: 10000)
- `--trade-size-max`: Maximum trade size in smallest units (default: 50000)
- `--enable-limit-order`: Enable limit order trading (default: false)

### Example `.env` file

```bash
# Account
EVM_MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
```

**Note**: The `.env` file is automatically loaded if present. You can also pass values via command-line flags, which take precedence over environment variables.

