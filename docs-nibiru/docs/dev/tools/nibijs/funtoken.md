---
order: 6
---

# FunToken (NibiJS)

The FunToken mechanism on Nibiru facilitates the seamless connection of fungible tokens across different environments, such as ERC20 tokens and Bank Coins. This feature enhances interoperability, improves liquidity, and elevates the user experience. It also promotes ecosystem growth by bridging Ethereum-based protocols with Nibiru’s interchain assets.

The FunToken module on Nibiru Chain allows users to create and manage multiple fungible tokens using unique subdenominations. The creator of a token automatically gains "admin" privileges over it.

With `NibiJS`, users can perform the following operations on FunTokens:

- **Create FunToken**
- **Convert Coin to EVM**

All operations are executed using the `NibiruTxClient` class, which facilitates communication with the blockchain.

---

## Interacting with TokenFactory (TF)

To interact with FunTokens, you need to connect your wallet to a `NibiruTxClient`. This client enables the use of the `signAndBroadcast` method for signing transactions.

Below is an example of how to connect to a wallet using [Cosmos-Kit](https://docs.cosmology.zone/cosmos-kit). For additional wallet connection methods, see [this guide](./connect-wallet.md).

---

### Connect to a Wallet

Here’s how to establish a connection with Cosmos-Kit:

```javascript
import { NibiruTxClient, Testnet } from "@nibiruchain/nibijs";
import { useChain } from "@cosmos-kit/react";

const chain = Testnet(1); // Testnet chain configuration
const { getOfflineSigner, address } = useChain(chain.chainName); // Wallet connection details

const txClient = await NibiruTxClient.connectWithSigner(
  chain.endptTm, // RPC endpoint
  getOfflineSigner
);
```

### Create FunToken

To create a FunToken, you need a minimum balance of 10,000 NIBI, in addition to the transaction's gas fee. This amount is required as a Create FunToken Fee.

```javascript
import { EncodeObject } from "@cosmjs/proto-signing";
import { MsgCreateFunToken, MsgCreateFunTokenResponse } from "@nibiruchain/nibijs/dist/src/protojs/eth.evm.v1/tx";

const createFunToken = async () => {
  const denom = "tf/nibi1creator-address/subdenom"; // Replace with the actual denomination

  const msgs: EncodeObject[] = [
    {
      typeUrl: "/eth.evm.v1.MsgCreateFunToken",
      value: MsgCreateFunToken.fromPartial({
        fromBankDenom: denom,
        sender: address, // Your wallet address
      }),
    },
  ];

  const fee = "auto"; // Automatically calculate transaction fee
  const tx = await txClient.signAndBroadcast(address, msgs, fee);
  const result = MsgCreateFunTokenResponse.decode(tx.msgResponses[0].value);

  console.log("Created FunToken Response:", result);
};
```

### Convert Coin To Evm

Convert bank to its ERC20 representation and send to the EVM account adress

```javascript
import { EncodeObject } from "@cosmjs/proto-signing";
import { MsgConvertCoinToEvm, MsgConvertCoinToEvmResponse } from "@nibiruchain/nibijs/dist/src/protojs/eth.evm.v1/tx";

const convertCoinToEvmResponse = async () => {
  const denom = "tf/nibi1creator-address/subdenom"; // Replace with the actual denomination

  const msgs: EncodeObject[] = [
    {
      typeUrl: "/eth.evm.v1.MsgConvertCoinToEvm",
      value: MsgConvertCoinToEvm.fromPartial({
        toEthAddr: "0xRecipientEthAddress", // Replace with the recipient's Ethereum address
        sender: address, // Your wallet address
        bankCoin: { denom, amount: "1000" }, // Specify the Bank Coin denomination and amount
      }),
    },
  ];

  const fee = "auto"; // Automatically calculate transaction fee
  const tx = await txClient.signAndBroadcast(address, msgs, fee);
  const result = MsgConvertCoinToEvmResponse.decode(tx.msgResponses[0].value);

  console.log("Converted Coin to EVM Response:", result);
};

```

### Query FunToken Mapping

Retrieve details about a specific FunToken mapping:

```javascript
const token = "0xTokenAddressOrDenomination"; // Either the ERC20 contract address or the Bank Coin denomination
const queryFunTokenInfo = await txClient.nibiruExtensions.query.eth.funTokenMapping({ token });
console.log("FunToken Mapping:", queryFunTokenInfo);
```

## Related Pages

- [NibiJS Getting Started](./getting-started.md)
- [NibiJS Connecting wiht a wallet extension](./connect-wallet.md)
- [FunToken Mechanism](../../../evm/funtoken.md)
- [Bank Coins](../../../concepts/tokens/bank-coins.md)
- [ERC20 Tokens](../../../concepts/tokens/erc20.md)
- [Tokens on Nibiru](../../../concepts/tokens/index.md)
