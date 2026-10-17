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

From `evm/e2e/`:

```bash
just install
```

Run this recipe from the repository root. It installs the locked Bun
dependencies in both `evm/e2e` and `evm/e2e/passkey-sdk`, then generates the
Hardhat types.

### Configure environment in `.env` file

Use [env.sample](./.env_sample) as a reference.

```ini
JSON_RPC_ENDPOINT="http://127.0.0.1:8545"
MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
```

The `MNEMONIC` above is the same BIP-39 phrase used by
`x/cli/localnet.sh` to create the `validator` key on the
`nibiru-localnet-0` chain. The `account` used in tests
(`test/testdeps.ts`) is the EVM signer for that validator/dev account,
which is funded in genesis and used to deploy contracts, pay gas, and
fund other test wallets.

### Execute

```bash
cd evm/e2e
❯ just test
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
cd evm/e2e
# Optional: provide QX/QY (0x-prefixed 32-byte coords) to create the first account.
ENTRY_POINT=0xYourEntryPoint \
QX=0x... \
QY=0x... \
just deploy-passkey
```

### Run an ERC-4337 bundler against Nibiru RPC

Uses a Stackup-style Docker image with a temp config. Required env: `RPC_URL`, `ENTRY_POINT`, `CHAIN_ID`. Optional:
`PRIVATE_KEY` (bundler signer), `BUNDLER_PORT` (default 4337), `BUNDLER_IMAGE` (default
`ghcr.io/stackup-wallet/stackup-bundler:latest`).

```bash
cd evm/e2e
RPC_URL=http://127.0.0.1:8545 \
ENTRY_POINT=0x... \
CHAIN_ID=12345 \
just bundler
```

### Passkey ERC-4337 test coverage

The E2E test suite now exercises the passkey ERC-4337 flow end-to-end. During the run it:

- Builds the `passkey-sdk`, deploys a fresh `EntryPointV06` + `PasskeyAccountFactory`, and funds the dev bundler key.
- Starts a bundler on port `14437`: by default the lightweight `passkey-sdk/dist/local-bundler.js`; set
  `PASSKEY_BUNDLER_MODE=official` to instead build and start `evm/passkey-bundler/dist/index.js` (ensure `bun install`
  in `evm/passkey-bundler/` first).
- Executes the CLI passkey script against that bundler to prove a full user operation.

Ensure Bun and the package dependencies are installed with `just install` in `evm/e2e/` (and `bun install` in
`evm/passkey-bundler/` if using `PASSKEY_BUNDLER_MODE=official`). Ensure port `14437` is free before running `just test`
or `just test-e2e`.
