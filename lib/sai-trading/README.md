# Nibiru/Sai-Trading

Rough notes (for now)

## Plan

The four Wasm fixtures are pinned to `wasm-contracts/v1.42.0`. Their source
commit and the retained Solidity fixture hashes are recorded separately in
`artifacts-lock.toml`. Verify the Wasm files with:

```sh
(cd artifacts && sha256sum --check checksums.txt)
```

The deployer grants the predicted vault-token-minter address `tf_oper` before
instantiation so its tokenfactory create-denom message is authorized. The localnet
validator is already sudo root. Perp markets are configured at instantiation;
subsequent setup uses the v1.42 admin message format. The trader supplies an
explicit collateral amount, calls `close_trade`, and lists positions through
`list_trades_for_user` in pages of 100 scanned indices. Listing reads the pinned
v1.42 `user_trade_index` storage counter and continues across empty pages.
A mined EVM revert is reported as an error even when
the enclosing Cosmos transaction has code zero.

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

