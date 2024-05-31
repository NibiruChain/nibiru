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

 PASS  test/contract_infinite_loop_gas.test.js (8.617 s)
  Infinite loop gas contract
    ✓ should fail due to out of gas error (4152 ms)

 PASS  test/contract_send_nibi.test.js (16.977 s)
  Send NIBI from smart contract
    ✓ send nibi via "sendViaTransfer" method (4244 ms)
    ✓ send nibi via "sendViaSend" method (4239 ms)
    ✓ send nibi via "sendViaCall" method (4259 ms)

 PASS  test/erc20.test.js (8.845 s)
  ERC-20 contract tests
    ✓ send, balanceOf (8765 ms)

 PASS  test/basic_queries.test.js
  Basic Queries
    ✓ Simple transfer, balance check (4224 ms)

Test Suites: 4 passed, 4 total
Tests:       6 passed, 6 total
Snapshots:   0 total
Time:        38.783 s, estimated 50 s
Ran all test suites.

```
