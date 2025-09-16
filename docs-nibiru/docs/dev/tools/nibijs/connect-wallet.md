---
order: 2
---

# Connecting IBC Wallets

Learn how to connect IBC-comptabile wallets to your application  so that users
can interact with the Nibiru blockchain. This guide is designed for developers
who want to integrate Nibiru functionality into their apps. {synopsis}

## What You'll Acheive

After following this guide, you'll be able to: 

1. **Connect wallets to an application** to interact with the Nibiru
   blockchain.
2. **Use a `NibiruTxClient`** to perform basic operations like querying account balances and sending transactions 
3. **Invoke wasm smart contracts** on the Nibiru blockchain.

Below is a step-by-step guide on how to connect to wallets using the set of
wallet adapters built into `Cosmos-Kit`, such as Leap Wallet, Keplr, and
Cosmostation. Alternatively, generate and use a newly created random wallet for
development purposes.

## Use Your Signer to Make a NibiruTxClient

```js
import {
  NibiruTxClient,
  Chain,
  Testnet,
} from "@nibiruchain/nibijs"

/**
 * A "Chain" object exposes all endpoints for Nibiru node such as the
 * gRPC server, Tendermint RPC endpoint, and REST server.
 *
 * The most important one for nibijs is "Chain.endptTM", the Tendermint RPC.
 **/
const chain: Chain = Testnet() // Permanent testnet

// ------------------
// NibiruTxClient
// 
// Signer: ⚠️  We' get this from the wallet
let signer = await signerFromWallet() 
const txClient = await NibiruTxClient.connectWithSigner(
  CHAIN.endptTm,
  signer
)
```

With `NibiJS`, developers have access to a transaction client (`NibiruTxClient`),
which extends `SigningStargateClient` and functions as both a signing client and
a `SigningCosmWasmClient`. 

This class enables interaction with smart contracts on the Nibiru blockchain,
including deployment, instantiation, querying, and execution.

## IBC Wallets - Cosmos-Kit (Leap, Keplr, etc.)

To setup or explore `Cosmos-Kit` by cosmology, check out their [documentation](https://docs.cosmology.zone/cosmos-kit).

1. Import Required Modules:

```javascript
import {  
  NibiruTxClient, 
  Testnet, 
  Mainnet 
} from '@nibiruchain/nibijs';
import { useChain } from "@cosmos-kit/react";
```

2. Connect to Nibiru:

```javascript
const chain = Testnet(1);
const { status, address, isWalletConnected, getOfflineSigner } =
    useChain(chain.chainName);
const txClient = await NibiruTxClient.connectWithSigner(
  chain.endptTm, // RPC endpoint
  getOfflineSigner
);
```

## IBC Wallets - Keplr

To setup or explore `Cosmos-Kit` by cosmology, check out their [documentation](https://docs.cosmology.zone/cosmos-kit).

1. Define Keplr Chain Info

```javascript
const chainInfoMainnet = {
  chainId: "cataclysm-1", // Replace with "nibiru-testnet-1" for testnet
  chainName: "cataclysm-1", // Replace with "nibiru-testnet-1" for testnet
  rpc: "https://rpc.nibiru.fi", // Replace with testnet URL if needed
  rest: "https://lcd.nibiru.fi", // Replace with testnet URL if needed
  stakeCurrency: {
    coinDenom: "NIBI",
    coinMinimalDenom: "unibi",
    coinDecimals: 6,
  },
  bip44: {
    coinType: 118,
  },
  bech32Config: {
    bech32PrefixAccAddr: "nibi",
    bech32PrefixAccPub: "nibipub",
    bech32PrefixValAddr: "nibivaloper",
    bech32PrefixValPub: "nibivaloperpub",
    bech32PrefixConsAddr: "nibivalcons",
    bech32PrefixConsPub: "nibivalconspub",
  },
  currencies: [
    {
      coinDenom: "NIBI",
      coinMinimalDenom: "unibi",
      coinDecimals: 6,
    },
  ],
  feeCurrencies: [
    {
      coinDenom: "NIBI",
      coinMinimalDenom: "unibi",
      coinDecimals: 6,
      gasPriceStep: {
        low: 0.025,
        average: 0.05,
        high: 0.1,
      },
    },
  ],
};
```

2. Import Required Modules:

```javascript
import {  
  NibiruTxClient, 
} from '@nibiruchain/nibijs';
```

3. Connect to Nibiru:

```javascript
 // Ensure Keplr is installed
if (!window.getOfflineSigner || !window.keplr) {
  alert("Please install Keplr extension");
  return;
}
// Suggest the chain to Keplr
await window.keplr.experimentalSuggestChain(chainInfo);

// Enable Keplr
await window.keplr.enable(chainInfo.chainId);

// Get the offline signer from Keplr
const offlineSigner = window.getOfflineSigner(chainInfo.chainId);

const txClient = await NibiruTxClient.connectWithSigner(
  chainInfo.rpc, // RPC endpoint
  offlineSigner
);
```

## IBC Wallets - Random Wallet

Using a newly created random wallet

1. Import Required Modules:

```javascript
import { 
  newRandomWallet, 
  newSignerFromMnemonic, 
  NibiruTxClient, 
  Testnet, 
  Mainnet 
} from '@nibiruchain/nibijs';
```

2. Create a Wallet:

```javascript
// Create a new Nibiru wallet
const wallet = await newRandomWallet()
const [{ address }] = await wallet.getAccounts()
const signer = await newSignerFromMnemonic(wallet.mnemonic)
```

3. Connect to Nibiru:

```javascript
const chain = Testnet(1);
const txClient = await NibiruTxClient.connectWithSigner(
  chain.endptTm, // RPC endpoint
  signer
);
```

## Related Pages

- [Guide: Building Apps with NibiJS](./README.md)
- [NibiJS - Installation](./install.md)
