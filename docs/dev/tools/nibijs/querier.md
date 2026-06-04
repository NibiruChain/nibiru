---
order: 4
---

# Querying Blockchain Data

`NibiJS` includes a `NibiruQuerier` class for querying blockchain data. This class allows developers to query token balances, block details, oracle prices, staking delegations, and more. The `NibiruQuerier` makes accessing blockchain data straightforward, enabling developers to efficiently build dApps on the Nibiru blockchain. With `NibiruQuerier`, nearly all blockchain data can be queried. Below are some basic examples, but many more queries are possible. Note that these queries can also be accessed via `NibiruTxClient`.

## Getting Started with NibiruQuerier

First, import the necessary classes and establish a connection to the blockchain.

```javascript
import { Chain, NibiruQuerier, Testnet } from "@nibiruchain/nibijs";

const connectToQuerier = async () => {
  /**
   * A "Chain" object exposes all endpoints for a Nibiru node, such as the
   * gRPC server, Tendermint RPC endpoint, and REST server.
   *
   * The most important endpoint for nibijs is "Chain.endptTM", the Tendermint RPC.
   **/
  const chain: Chain = Testnet(1);

  const querier = await NibiruQuerier.connect(chain.endptTm);
  return querier;
};
```

## Querying Token Balances

### All Token Balances

Retrieve all token balances for a given address.

```javascript
const queryAllBalances = async (querier, address) => {
  const allBalances = await querier.getAllBalances(address);
  console.log("All Balances:", allBalances);
  return allBalances;
};

// Usage
const querier = await connectToQuerier();
const address = "your_nibi_address"; // Replace with your Nibiru address
const allBalances = await queryAllBalances(querier, address);
```

### Single Token Balance

Retrieve the balance of a specific token for a given address.

```javascript
const queryBalance = async (querier, address, denom) => {
  const balance = await querier.getBalance(address, denom);
  console.log(`${denom} Balance:`, balance);
  return balance;
};

// Usage
const querier = await connectToQuerier();
const address = "your_nibi_address"; // Replace with your Nibiru address
const balance = await queryBalance(querier, address, "unibi");
```

## Querying Block Information

### Latest Block Height

Get the height of the latest block.

```javascript
const queryLatestHeight = async (querier) => {
  const latestHeight = await querier.getHeight();
  console.log("Latest Block Height:", latestHeight);
  return latestHeight;
};

// Usage
const querier = await connectToQuerier();
const latestHeight = await queryLatestHeight(querier);
```

### Block Details by Height

Get details of a block by its height.

```javascript
const queryBlockByHeight = async (querier, height) => {
  const block = await querier.tm.block(height);
  console.log("Block Details:", block);
  return block;
};

// Usage
const querier = await connectToQuerier();
const latestHeight = await queryLatestHeight(querier);
const block = await queryBlockByHeight(querier, latestHeight);
```

### Latest Block Details

Get details of the latest block.

```javascript
const queryLatestBlock = async (querier) => {
  const latestBlock = await querier.getBlock();
  console.log("Latest Block Details:", latestBlock);
  return latestBlock;
};

// Usage
const querier = await connectToQuerier();
const latestBlock = await queryLatestBlock(querier);
```

## Advanced Queries with Nibiru Extensions

### Oracle Price Exchange Rate

Get the exchange rate between UNIBI and UUSDT.

```javascript
const queryOraclePrice = async (querier, pair) => {
  const price = await querier.nibiruExtensions.oracle.exchangeRate(pair);
  console.log(`Oracle Price (${pair}):`, price);
  return price;
};

// Usage
const querier = await connectToQuerier();
const oraclePrice = await queryOraclePrice(querier, "unibi:uusdt");
```

### All IBC Channels

Retrieve all IBC channels.

```javascript
const queryAllIBCChannels = async (querier) => {
  const channels = await querier.nibiruExtensions.ibc.channel.allChannels();
  console.log("All IBC Channels:", channels);
  return channels;
};

// Usage
const querier = await connectToQuerier();
const allIBCChannels = await queryAllIBCChannels(querier);
```

### Staking Delegations

Get all staking delegations for a given address.

```javascript
const queryDelegations = async (querier, address) => {
  const delegations = await querier.nibiruExtensions.staking.delegatorDelegations(address);
  console.log("Delegations:", delegations);
  return delegations;
};

// Usage
const querier = await connectToQuerier();
const address = "your_nibi_address"; // Replace with your Nibiru address
const delegations = await queryDelegations(querier, address);
```

### Unbonding Staking Delegations

Get all unbonding staking delegations for a given address.

```javascript
const queryUnbondingDelegations = async (querier, address) => {
  const unbondingDelegations = await querier.nibiruExtensions.staking.delegatorUnbondingDelegations(address);
  console.log("Unbonding Delegations:", unbondingDelegations);
  return unbondingDelegations;
};

// Usage
const querier = await connectToQuerier();
const address = "your_nibi_address"; // Replace with your Nibiru address
const unbondingDelegations = await queryUnbondingDelegations(querier, address);
```

## Related Pages

- [NibiJS Getting Started](./getting-started.md)
- [NibiJS Connecting wiht a wallet extension](./connect-wallet.md)
