# EVM Tests

Folder contains ethers.js test bundle which executes main
Nibiru EVM methods via JSON RPC.

Contract [TestERC20.sol](./contracts/TestERC20.sol) represents
simple ERC20 token with initial supply `1000,000 * 10e18` tokens.

Contract is compiled via HardHat into [json file](./contracts/TestERC20Compiled.json)
with ABI and bytecode.

## Setup and Run

### Run Nibiru node

Tests require Nibiru node running with JSON RPC enabled.

Localnet has JSON RPC enabled by default.

### Install dependencies

```bash
npm install
```

### Configure environment in `.env` file

Use [env.sample](./.env_sample) as a reference.

```ini
JSON_RPC_ENDPOINT="http://127.0.0.1:8545"
MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
```

### Execute

```bash
❯ bun test
bun test v1.1.12 (43f0913c)

test/erc20.test.ts:
✓ ERC-20 contract tests > should send properly [8410.41ms]

test/native_transfer.test.ts:
✓ Basic Queries > Simple transfer, balance check [4251.02ms]

test/contract_infinite_loop_gas.test.ts:
✓ Infinite loop gas contract > should fail due to out of gas error [8281.13ms]

test/contract_send_nibi.test.ts:
✓ Send NIBI via smart contract > should send via transfer method [4228.80ms]
✓ Send NIBI via smart contract > should send via send method [4231.87ms]
✓ Send NIBI via smart contract > should send via transfer method [4213.43ms]

 6 pass
 0 fail
 22 expect() calls
Ran 6 tests across 4 files. [38.08s]

```

### Deploy PasskeyAccountFactory on Nibiru localnet

Assumes:
- Nibiru localnet running with the P-256 precompile at `0x0000000000000000000000000000000000000100`.
- An ERC-4337 EntryPoint already deployed; pass its address via `ENTRY_POINT`.

```bash
# Optional: provide QX/QY (0x-prefixed 32-byte coords) to create the first account.
ENTRY_POINT=0xYourEntryPoint \
QX=0x... \
QY=0x... \
npx hardhat run scripts/deploy-passkey.js --network localhost
```

### Run an ERC-4337 bundler against Nibiru RPC

Uses a Stackup-style Docker image with a temp config. Required env: `RPC_URL`, `ENTRY_POINT`, `CHAIN_ID`. Optional:
`PRIVATE_KEY` (bundler signer), `BUNDLER_PORT` (default 4337), `BUNDLER_IMAGE` (default
`ghcr.io/stackup-wallet/stackup-bundler:latest`).

```bash
RPC_URL=http://127.0.0.1:8545 \
ENTRY_POINT=0x... \
CHAIN_ID=12345 \
npm run bundler
```
