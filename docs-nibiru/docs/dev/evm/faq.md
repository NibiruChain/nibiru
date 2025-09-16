---
order: 5
metaTitle: "Nibiru EVM | FAQ"
---

# Frequently Asked Questions

---

### 🔍 What is the EVM Explorer for Nibiru?

**Mainnet:**  
📎 [https://nibiscan.io](https://nibiscan.io)

**Testnet (Testnet-2):**  
📎 [https://testnet.nibiscan.io](https://testnet.nibiscan.io)

---

### 🌐 What are the EVM RPC URLs and Chain IDs?

**Mainnet**  

- RPC: `https://evm-rpc.nibiru.fi`  
- Chain ID: `6900` (Hex: `0x1AF4`)  
- Chain Name: `cataclysm-1`

**Testnet-2**  

- RPC: `https://evm-rpc.testnet-2.nibiru.fi`  
- Chain ID: `6911` (Hex: `0x1AFF`)  
- Chain Name: `nibiru-testnet-2`

---

### 💡 What are FunTokens?

FunTokens enable seamless movement of tokens between the EVM and Cosmos environments on Nibiru.  
You can:

- Convert ERC-20 tokens into Cosmos bank coins
- Convert Cosmos coins into ERC-20 tokens

This powers composability across DeFi apps, IBC, and Wasm smart contracts — all in one ecosystem.

---

### 🔄 How do I convert ERC-20 to Cosmos bank coins?

Call the **FunToken precompile** at `0x000...0800` using `sendToBank()` and specify your Cosmos Bech32 address.  
It’ll convert your ERC-20 tokens to Bank Coins.

---

### 🔁 How do I convert Cosmos coins to ERC-20?

Use `sendToEvm()` on the FunToken precompile.  
It burns your Cosmos coins and releases (or mints) ERC-20 to your Ethereum address.

---

### 📦 What are Precompiles on Nibiru?

Nibiru includes special precompiled contracts for cross-VM interoperability:

- **FunToken Precompile** `0x000...0800`: ERC-20 ↔ Bank coin transfer
- **Oracle Precompile** `0x000...0801`: Live price feeds (ChainLink format)
- **Wasm Precompile** `0x000...0802`: Call/query Wasm contracts from Solidity

These enable native bridging and modular contract design without external dependencies.

---

### 📊 How can I access live prices from oracles?

Use the **Oracle Precompile** at `0x0000000000000000000000000000000000000801`.  
It supports ChainLink’s `AggregatorV3Interface`, so you can fetch prices like `ETH/USD`, `BTC/USD`, `NIBI/USD` in Solidity, or via Ethers.js.

---

### 🧰 How do I deploy Solidity contracts to Nibiru?

You can use **Hardhat**:

```bash
npx hardhat run scripts/deploy.js --network nibiru
```

Example config:

```js
networks: {
  nibiru: {
    url: "https://evm-rpc.nibiru.fi",
    chainId: 6900,
    accounts: [PRIVATE_KEY],
  }
}
```

---

### 📚 How do I compile smart contracts for Nibiru?

Use Hardhat with Solidity 0.8.19+ and run:

```bash
npx hardhat compile
```

Make sure your `hardhat.config.js` includes:

```js
require("@nomicfoundation/hardhat-toolbox");

module.exports = {
  solidity: "0.8.19",
};
```

---

### 📈 What's on the roadmap?

Upcoming features include:

- Multi-VM composability (Wasm + EVM)
- Expanded oracle feeds and precompile utilities
- Bridged and native stablecoins

📅 [Roadmap page →](https://nibiru.fi/docs/ecosystem/future)

---

### 🧪 Is there a faucet for Testnet?

Yes — use the official faucet at:  
🌐 [https://app.nibiru.fi/faucet](https://app.nibiru.fi/faucet).

---

### 🛠 What tools and SDKs are available?

- [`@nibiruchain/evm-core`](https://www.npmjs.com/package/@nibiruchain/evm-core) – Chain utilities & ABI helpers
- [`@nibiruchain/solidity`](https://www.npmjs.com/package/@nibiruchain/solidity) – ABI bindings for precompiles
- [`NibiruJS`](https://github.com/NibiruChain/nibijs) – Wasm + GraphQl SDK

---

### 📘 Is Wasm supported on Nibiru?

Yes. You can deploy and interact with CosmWasm contracts.  
EVM developers can call Wasm contracts directly using the **Wasm Precompile** (`0x000...0802`).

---

### 🌉 How do I bridge tokens into Nibiru?

For now, bridging is manual via IBC for test assets. Nibiru is working on integrations with bridge providers like **Axelar**, **IBC**, and **LayerZero** for both testnet and mainnet.

---

### 🪙 Does Nibiru have a native token?

Yes — the native token is **$NIBI**. It’s used for:

- Staking and governance
- Paying gas fees on both Cosmos & EVM layers
- Collateral and incentive systems
- **WNIBI** is the ERC20 version is NIBI called Wrapped NIBI

---

### 📤 Can I launch a token (ERC20 or IBC) on Nibiru?

Yes. You can:

- Use `ERC20Minter.sol` to deploy custom tokens via Hardhat
- Use `MsgCreateFunToken` to create a Cosmos-EVM mapped token

---

### 🪙 Does Nibiru support token standards like CW20, ERC20, etc.?

Yes — Nibiru supports:

- **ERC20** on the EVM layer
- **Bank** native and Token Factory tokens on the Wasm Layer
- **CW20/CW721** on the Wasm layer
- **ICS-20** for IBC tokens

And via **FunToken**, you can map these across VMs!

---

### ⚒ How do I test smart contract interactions?

You can:

- Use **Hardhat + Ethers.js** on the EVM side
- Use ***NibiJS** on the Cosmos/Wasm side
- Call Wasm contracts from Solidity via the Wasm precompile

---

### ⚡ What’s the transaction speed and throughput?

- Nibiru processes up to **10,000+ TPS**
- EVM layer supports **single-threaded execution** now, with **parallel optimistic execution** (PARE) on the roadmap

---

### 🧑‍💼 How can projects get listed or supported by Nibiru?

Submit your project or partnership request via the [Ecosystem Portal](https://nibiru.fi/ecosystem) or reach out in [Discord](https://discord.gg/nibiru). There's also a grants program underway.

---

### 🧠 Is there indexer or subgraph support?

Not yet out of the box, but:

- EVM-compatible indexers like [**SubQuery**](https://github.com/subquery/ethereum-subql-starter/tree/main/Nibiru?ref=blog.subquery.network) can work

---

### 🏗 Can I use MetaMask?

Yes — Nibiru EVM is MetaMask compatible.  
Just add the RPC, chain ID, and you're good to go.

Example config for MetaMask:

```json
{
  "network": "Nibiru Mainnet",
  "rpc": "https://evm-rpc.nibiru.fi",
  "chainId": 6900,
  "currency": "NIBI",
  "explorer": "https://nibiscan.io"
}
```

---

### 🔗 Where can I learn more?

- 📚 [Docs](https://nibiru.fi/docs)  
- 💬 [Discord](https://discord.gg/nibiru)  
- 🧑‍💻 [GitHub](https://github.com/NibiruChain)  
- 🐦 [Twitter / X](https://twitter.com/NibiruChain)
