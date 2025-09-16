---
order: 1
metaTitle: "ERC20 Tokens | Fungible Token Standards on Nibiru"
---

# ERC20 Tokens

With the development of Nibiru EVM, the network supports Ethereum smart
contracts, allowing for the deployment of ERC20 tokens. Each ERC20 token is a
fungible digital asset on the Nibiru blockchain. {synopsis}


<!-- Outline 


- [ ] Create guide on making an ERC20 and registering it to have a FunToken
mapping, opening it up to use as an interchain asset
-->

:::tip
New to ERC20 tokens? [Go to this section](#erc20-token-standard) to learn more.
:::

## FunToken Mapping Mechanism

How this works is that a canonical mapping called a "FunToken mapping" can be
created between a ERC20 and Bank Coin. This mapping ties a unique connection
between the two fungible token representations and enables all accounts to freely
convert between both forms.

## How do Bank Coins differ from ERC20 tokens?

| Aspect | Bank Coins | ERC20 Tokens |
|--------|------------|----------------|
| State Management | Managed directly by the Bank module in its own key-value store | Managed within each token's smart contract state |
| Creation Process | Created through governance proposals, permissioned modules, or the Tokenfactory module | Created by deploying new smart contracts |
| Efficiency | Typically more gas-efficient due to native implementation | Generally consume more gas due to smart contract execution |
| Standardization | All follow the same standard and are handled uniformly by the blockchain | Can have variations in implementation, though most follow a standard interface |
| Balance Queries | Queried through the Bank module's API, which is easy to manage because the same generic query is used for all coins. | Queried by calling the ERC20 token's smart contract, as balances are individually stored on each contract. |
| Transfer Mechanism | Executed directly by the Bank module | Executed by calling the token's transfer function |
| Supply Management | Controlled by authorized modules or governance | Managed by the token's smart contract logic |
| Permissionless Creation | Possible via the Tokenfactory module | Inherent (anyone can deploy a new token contract) |
| Storage Efficiency | More efficient as all coins share the same infrastructure | Each token contract stores its own data separately |
| Interoperability | Native to the Bank Module but can be converted to ERC20 via the FunToken system | Native to EVM environments (Nibiru EVM in this case) but can be converted to Bank Coins via the FunToken system |

## Bridged USDC Standard

Circle's Bridged USDC Standard will be deployed on the Nibiru EVM when V2 hits
mainnet. 

> "Bridged USDC Standard is a specification and process for deploying a bridged
> form of USDC on EVM blockchains with optionality for Circle to seamlessly
> upgrade to native issuance in the future.
> 
> The result is a secure and standardized way for any EVM blockchain and rollup
> team to transfer ownership of a bridged USDC token contract to Circle to
> facilitate an upgrade to native USDC, if and when both parties deem
> appropriate." - [Circle Blog](https://www.circle.com/blog/bridged-usdc-standard).

More information on this is coming soon.

<!-- 
TODO: 
- [ ] Include Bridged USDC address
- [ ] Add link to guide on Bridged USDC on Nibiru EVM
--> 

## LayerZero Assets

More information on this is coming soon.

## ERC20 Token Standard

ERC20, introduced in 2015 as part of an Etheruem Improvement Proposal (EIP-20), is the most widely adopted standard for fungible tokens on Ethereum-compatible blockchains, such as Nibiru. The ERC20 standard defines a set of rules that tokens must adhere to, ensuring compatibility with wallets, dApps, and other blockchain tools.

At its core, each ERC20 token is implemented as a smart contract—self-executing code on the blockchain—that handles all token functionality, including managing balances, transfers, and allowances.

The ERC20 standard specifies several core functions and events, which include:

- `balanceOf(address owner)`: Returns the balance of tokens for the given address.
- `transfer(address to, uint256 value)`: Moves tokens to another address.
- `approve(address spender, uint256 value)`: Sets an allowance for another address to spend tokens.
- `allowance(address owner, address spender)`: Returns the remaining number of tokens a spender can transfer on behalf of the owner.
- `transferFrom(address from, address to, uint256 value)`: Transfers tokens from one address to another using an allowance.
- `totalSupply()`: Returns the total number of tokens in existence.

These functions allow apps to interact with ERC20 tokens in a standardized manner.
