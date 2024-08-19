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

test/basic_queries.test.ts:
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
