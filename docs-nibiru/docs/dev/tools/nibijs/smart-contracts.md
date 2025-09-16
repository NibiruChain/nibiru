---
order: 3
---

# Smart Contracts (Wasm)

With `NibiJS`, developers can utilize the `SigningCosmWasmClient` within the `NibiruTxClient` class to interact with a smart contract. This allows them to upload the contract's bytecode, set the sender address, specify the fee, and deploy the contract to the blockchain. Additionally, it simplifies the processes of smart contract instantiation, querying, and execution.

## Interacting with Smart Contracts

To interact with smart contracts, you need to connect your wallet to a `NibiruTxClient`. This provides access to `SigningCosmWasmClient` via the `wasm` property. In the example below, we are connecting via [Cosmos-Kit](https://docs.cosmology.zone/cosmos-kit). For other options, check out [this guide](./connect-wallet.md).

### Connect to a Wallet

First, set up the connection to your wallet. This example is using Cosmos-Kit.

```javascript
import {
  NibiruTxClient,
  Testnet,
} from "@nibiruchain/nibijs"
import { useChain } from "@cosmos-kit/react";

const chain = Testnet(1);
const { getOfflineSigner } = useChain(chain.chainName);
const txClient = await NibiruTxClient.connectWithSigner(
  chain.endptTm, // RPC endpoint
  getOfflineSigner
);
```

### Deploying Smart Contract

To deploy a smart contract, you need to upload the WASM bytecode to the blockchain.

```javascript
const senderAddress = "your_sender_address"; // Replace with your sender address
const wasmCode = fs.readFileSync("path_to_your_wasm_file.wasm"); // Read the WASM file as a Uint8Array
const fee = "auto"; // You can specify the fee if needed
const deployContract = await txClient.wasmClient.upload(
  senderAddress, 
  wasmCode, 
  fee
);
```

### Instantiating a Smart Contract

To query a smart contract, provide the contract address and the query message.

After deploying the contract, instantiate it by providing the codeId and initialization parameters.

```javascript
const senderAddress = "your_sender_address"; // Replace with your sender address
const codeId = deployContract.codeId; // Use the code ID from the deployment step

const instantiateMsg = {}; // Replace with the contract's instantiate message
const label = "My Smart Contract"; // A human-readable label for your contract
const fee = "auto"; // You can specify the fee if needed
const instantiateContract = await txClient.wasmClient.instantiate(
  senderAddress,
  codeId,
  instantiateMsg,
  label,
  fee
);
```

### Querying a Smart Contract

To query a smart contract, provide the contract address and the query message.

```javascript
const contractAddress = "your_contract_address"; // Replace with your contract address
const queryMsg = {}; // Replace with the query message for your contract
const queryContract = await txClient.wasmClient.queryContractSmart(
  contractAddress, 
  queryMsg
);
```

### Executing a Smart Contract

To execute a smart contract, provide the sender address, contract address, and execution message.

```javascript
const senderAddress = "your_sender_address"; // Replace with your sender address
const contractAddress = "your_contract_address"; // Replace with your contract address
const executeMsg = {}; // Replace with the execution message for your contract
const fee = "auto"; // You can specify the fee if needed
const executeContract = await txClient.wasmClient.execute(
  senderAddress,
  contractAddress,
  executeMsg,
  fee
);
```

## Related Pages

- [NibiJS Getting Started](./getting-started.md)
- [NibiJS Connecting wiht a wallet extension](./connect-wallet.md)
