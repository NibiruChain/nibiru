# @nibiruchain/evm-core

Core Nibiru EVM TypeScript library with functions to call Nibiru-specific precompiled contracts.

## Install

```bash
yarn add @nibiruchain/evm-core
# OR npm install OR bun install
```

## Usage with ethers

The `@nibiruchain/evm-core/ethers` subdirectory exports strongly typed
`ethers.Contract` objects for each precompiled contract, each available from
simple contstructors.

```bash
yarn add @nibiruchain/evm-core ethers@6
```

```js
import {
  funtokenPrecompile,
  wasmPrecompile,
} from "@nibiruchain/evm-core/ethers";
import { ethers } from "ethers";

// Set up ethers v6 provider. Most apps will use a Browser provider based on the
// window.ethereum object.
const provider = ethers.BrowserProvider(window.ethereum);
const signer = await provider.getSigner();

const wasmCaller = wasmPrecompile(provider)
// const wasmCaller = wasmPrecompile(account)
// NOTE: Both wallets and providers are valid ethers.ContractRunner instances,
// meaning both `account` and `provider` local variables make sense.

// Available methods
wasmCaller.execute;
wasmCaller.executeMulti;
wasmCaller.query;
wasmCaller.queryRaw;
wasmCaller.execute;
```

You can use the code below to set up a provider outside of a browser context.
```bash
yarn add dotenv ethers@6
```

```js
import { config } from "dotenv";
import { ethers, Wallet } from "ethers";
const provider = new ethers.JsonRpcProvider(process.env.JSON_RPC_ENDPOINT);
const account = Wallet.fromPhrase(process.env.MNEMONIC!, provider);
```

## ABIs and Precompile Addresses

We export constants for the address of each precompiled contract:

```js
import {
  ADDR_WASM_PRECOMPILE,
  ADDR_FUNTOKEN_PRECOMPILE,
  ADDR_ORACLE_PRECOMPILE,
} from "@nibiruchain/evm-core";
```

ABI objects can be imported as well.

```js
import {
  ABI_WASM_PRECOMPILE,
  ABI_FUNTOKEN_PRECOMPILE,
  ABI_ORACLE_PRECOMPILE,
} from "@nibiruchain/evm-core";
```
