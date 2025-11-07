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

