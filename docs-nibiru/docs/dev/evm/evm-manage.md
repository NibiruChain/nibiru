---
order: 3
metaTitle: "Nibiru EVM | Interacting with EVM Smart Contracts"
---

# Interacting with EVM Smart Contracts

This guide will help you understand how to compile, deploy, and interact with Solidity smart contracts on the Nibiru EVM using `ethers.js`.{synopsis}

## Deploying a Smart Contract

One way to deploy smart contracts on Nibiru EVM is by using the `ethers` npm library. You'll need a mnemonic phrase and the EVM RPC endpoint of your Nibiru chain (e.g., localnet is [127.0.0.1:8545](http://127.0.0.1:8545)).

```javascript
import { ethers } from "ethers"
import { config } from "dotenv"
import * as fs from "fs"

config()

const rpcEndpoint = process.env.JSON_RPC_ENDPOINT
const mnemonic = process.env.MNEMONIC

const provider = ethers.getDefaultProvider(rpcEndpoint)
const wallet = ethers.Wallet.fromPhrase(mnemonic)
const account = wallet.connect(provider)

const deployContract = async (path: string) => {
  const contractJSON = JSON.parse(
    fs.readFileSync(`contracts/${path}`).toString(),
  )
  const bytecode = contractJSON["bytecode"]
  const abi = contractJSON["abi"]

  const contractFactory = new ethers.ContractFactory(abi, bytecode, account)
  const contract = await contractFactory.deploy()
  await contract.waitForDeployment()
  return contract
}
```

This script performs the following steps:

- Imports necessary libraries and reads environment variables.
- Sets up the RPC provider and wallet using the mnemonic phrase.
- Reads the compiled contract's JSON file to get the bytecode and ABI.
- Uses the ContractFactory to deploy the contract and waits for the deployment to complete.

## Environment Setup

Ensure you have a .env file with the following variables:

```bash
JSON_RPC_ENDPOINT=http://127.0.0.1:8545
MNEMONIC=your-mnemonic-phrase-here
```

## Basic Queries

Once your contract is deployed, you can interact with it using ethers.js. Here are some basic queries you might perform:

```javascript
import { ethers } from "ethers";

const rpcEndpoint = process.env.JSON_RPC_ENDPOINT
const mnemonic = process.env.MNEMONIC

const provider = ethers.getDefaultProvider(rpcEndpoint)
const wallet = ethers.Wallet.fromPhrase(mnemonic)
const account = wallet.connect(provider)

const randomAddress = ethers.Wallet.createRandom().address;
const amountToSend = ethers.utils.parseUnits("1000", "wei");
const gasLimit = 100000; // Example gas limit

const getBalances = async (address) => {
  const balance = await provider.getBalance(address);
  return ethers.utils.formatUnits(balance, "wei");
};

const transferEthers = async () => {
  const senderBalanceBefore = await getBalances(account.address);
  const recipientBalanceBefore = await getBalances(randomAddress);

  console.log(`Sender balance before: ${senderBalanceBefore}`);
  console.log(`Recipient balance before: ${recipientBalanceBefore}`);

  // Execute EVM transfer
  const transaction = {
    to: randomAddress,
    value: amountToSend,
    gasLimit: gasLimit,
  };
  const txResponse = await account.sendTransaction(transaction);
  await txResponse.wait();

  const senderBalanceAfter = await getBalances(account.address);
  const recipientBalanceAfter = await getBalances(randomAddress);

  console.log(`Sender balance after: ${senderBalanceAfter}`);
  console.log(`Recipient balance after: ${recipientBalanceAfter}`);
};

transferEthers();
```

## Executing Smart Contract Functions

To execute functions within your deployed smart contract, you can use `ethers.js` to create a contract instance and call its methods.

### Example: Calling a Read-Only Function

Assuming you have a simple smart contract with a getValue function:

```javascript
// SimpleStorage.sol
pragma solidity ^0.8.0;

contract SimpleStorage {
    uint256 private value;

    // Function to set a new value. This modifies the blockchain state.
    function setValue(uint256 newValue) public {
        value = newValue;
    }

    // Function to get the current value. This does not modify the blockchain state.
    function getValue() public view returns (uint256) {
        return value;
    }
}
```

Here’s how you can call the getValue function:

```javascript
import { ethers } from "ethers";
import SimpleStorageABI from "./SimpleStorage.json"; // Assuming ABI is in JSON format
import { config } from "dotenv";

// Load environment variables from .env file
config();

// Set up the RPC endpoint and mnemonic phrase from environment variables
const rpcEndpoint = process.env.JSON_RPC_ENDPOINT;
const mnemonic = process.env.MNEMONIC;

// Connect to the Ethereum network using the RPC endpoint
const provider = new ethers.JsonRpcProvider(rpcEndpoint);

// Create a wallet instance using the mnemonic phrase
const wallet = ethers.Wallet.fromMnemonic(mnemonic);

// Connect the wallet to the provider
const account = wallet.connect(provider);

// Address of the deployed SimpleStorage contract
const contractAddress = "your_contract_address_here";

// Create a contract instance using the contract address, ABI, and provider
const simpleStorage = new ethers.Contract(contractAddress, SimpleStorageABI, provider);

// Function to call the read-only getValue function of the contract
const getValue = async () => {
  // Call the getValue function
  const value = await simpleStorage.getValue();
  
  // Log the returned value
  console.log(`Stored value: ${value}`);
};

// Execute the getValue function
getValue();
```

## Example: Calling a State-Changing Function

To call a function that changes the contract’s state (e.g., setValue):

```javascript
// Function to call the state-changing setValue function of the contract
const setValue = async (newValue) => {
  // Connect the contract instance to the wallet for signing transactions
  const contractWithSigner = simpleStorage.connect(account);

  // Call the setValue function with the new value
  const txResponse = await contractWithSigner.setValue(newValue);

  // Wait for the transaction to be mined
  await txResponse.wait();

  // Log the new value that was set
  console.log(`New value set: ${newValue}`);
};

// Execute the setValue function with a new value
setValue(42);
```

## Sending Funds from a Smart Contract

Given the following ERC20 FunToken.sol contract:

```javascript
pragma solidity ^0.8.24;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract FunToken is ERC20 {
    // Define the initial supply of FunToken: 1,000,000 tokens
    uint256 constant initialSupply = 1000000 * (10**18);

    // Constructor will be called on contract creation
    constructor() ERC20("FunToken", "FUN") {
        // Mint the initial supply to the deployer's address
        _mint(msg.sender, initialSupply);
    }
}
```

After deploying your contract via the deploy function [above](./evm-manage.md#deploying-a-smart-contract),

```javascript
import { deployContract } from "./deploy"; // Import the deploy function
import { ethers } from "ethers";
import FunTokenCompiled from "./FunTokenCompiled.json"; // Assuming ABI and bytecode are in JSON format
import { config } from "dotenv";

// Load environment variables from .env file
config();

// Set up the RPC endpoint and mnemonic phrase from environment variables
const rpcEndpoint = process.env.JSON_RPC_ENDPOINT;
const mnemonic = process.env.MNEMONIC;

// Connect to the Ethereum network using the RPC endpoint
const provider = new ethers.JsonRpcProvider(rpcEndpoint);

// Create a wallet instance using the mnemonic phrase
const wallet = ethers.Wallet.fromMnemonic(mnemonic);

// Connect the wallet to the provider
const account = wallet.connect(provider);

// Main function to deploy and interact with the FunToken contract
const main = async () => {
  // Deploy the FunToken contract and get the contract instance
  const contract = (await deployContract("FunTokenCompiled.json")) as ethers.Contract;
  
  // Get the address of the deployed contract
  const contractAddress = contract.address;
  console.log(`FunToken deployed at: ${contractAddress}`);

  // Generate a random address to receive tokens
  const shrimpAddress = ethers.Wallet.createRandom().address;
  
  // Define the amount of tokens to send (1,000 tokens)
  const amountToSend = ethers.utils.parseUnits("1000", 18);

  // Get the initial balances of the owner and the recipient
  let ownerBalance = await contract.balanceOf(account.address);
  let shrimpBalance = await contract.balanceOf(shrimpAddress);

  // Log the initial balances
  console.log(`Owner balance before: ${ethers.utils.formatUnits(ownerBalance, 18)}`);
  console.log(`Shrimp balance before: ${ethers.utils.formatUnits(shrimpBalance, 18)}`);

  // Connect the contract instance to the wallet for signing transactions
  const contractWithSigner = contract.connect(account);

  // Transfer tokens from the owner to the recipient
  const tx = await contractWithSigner.transfer(shrimpAddress, amountToSend);
  
  // Wait for the transaction to be mined
  await tx.wait();

  // Get the updated balances of the owner and the recipient
  ownerBalance = await contract.balanceOf(account.address);
  shrimpBalance = await contract.balanceOf(shrimpAddress);

  // Log the updated balances
  console.log(`Owner balance after: ${ethers.utils.formatUnits(ownerBalance, 18)}`);
  console.log(`Shrimp balance after: ${ethers.utils.formatUnits(shrimpBalance, 18)}`);
};

// Execute the main function
main();
```

## Related Pages

- [Nibiru EVM](../../evm/README.md)
- [Nibiru Networks](../networks)
- [Nibiru Source Code Installation Guide](../cli/nibid-binary.md#install-option-3--building-from-the-source-code)
