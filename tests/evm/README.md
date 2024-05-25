# EVM Tests

Folder contains ethers.js test bundle which executes main
Nibiru EVM methods via JSON RPC. 

Contract [FunToken.sol](./contracts/FunToken.sol) represents
simple ERC20 token with initial supply `1000,000 * 10e18` tokens.

Contract is compiled via HardHat into [json file](./contracts/FunTokenCompiled.json)
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
npm test

> nibiru-evm-test@0.0.1 test
> jest

 PASS  test/evm.test.js (13.163 s)
  Ethereum JSON-RPC Interface Tests
    ✓ Simple Transfer, balance check (4258 ms)
    ✓ Smart Contract (8656 ms)

Test Suites: 1 passed, 1 total
Tests:       2 passed, 2 total
Snapshots:   0 total
Time:        13.187 s, estimated 14 s
Ran all test suites.
```
