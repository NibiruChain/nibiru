---
order: 5
---

# Creating Fungible Tokens (NibiJS)

The TokenFactory (TF) module on Nibiru Chain allows any account to create new tokens with a specific format. This format includes the creator's address and a subdenomination, enabling token creation without requiring prior permissions and avoiding naming conflicts.

Using this module, a single account can create multiple tokens by providing unique subdenominations for each one. The creator of a denomination automatically gains "admin" privileges over the token.

With `NibiJS`, users can perform the following operations on TokenFactory tokens:

- **Create denominations**
- **Mint tokens**
- **Burn tokens**
- **Change token administrators**

All interactions utilize the `NibiruTxClient` class, which facilitates communication with the blockchain.

---

## Interacting with TokenFactory (TF)

To interact with TokenFactory, you first need to connect your wallet to a `NibiruTxClient`. This client provides access to the `signAndBroadcast` method to sign transactions.

Below, we demonstrate connecting to a wallet using [Cosmos-Kit](https://docs.cosmology.zone/cosmos-kit). For other wallet connection methods, refer to [this guide](./connect-wallet.md).

---

### Connect to a Wallet

Here is how to establish a connection with Cosmos-Kit:

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

### Create Denomination

Use the following code to create a new denomination. Replace the `subdenom` with your desired unique identifier.

```javascript
import { EncodeObject } from "@cosmjs/proto-signing";
import { MsgCreateDenom, MsgCreateDenomResponse } from "@nibiruchain/nibijs/dist/src/protojs/nibiru/tokenfactory/v1/tx";

const createDenom = async () => {
  const msgs: EncodeObject[] = [
    {
      typeUrl: "/nibiru.tokenfactory.v1.MsgCreateDenom",
      value: MsgCreateDenom.fromPartial({
        sender: address, // Your wallet address
        subdenom: "utest", // Replace with your subdenomination
      }),
    },
  ];

  const fee = "auto"; // Automatically calculate transaction fee
  const tx = await txClient.signAndBroadcast(address, msgs, fee);
  const result = MsgCreateDenomResponse.decode(tx.msgResponses[0].value);

  console.log("Created Denom Response:", result);
};
```

### Mint Tokens

This example mints tokens for the specified `denom` and sends them to a given address.

```javascript
import { EncodeObject } from "@cosmjs/proto-signing";
import { MsgMint, MsgMintResponse } from "@nibiruchain/nibijs/dist/src/protojs/nibiru/tokenfactory/v1/tx";

const mint = async () => {
  const denom = "tf/nibi1creator-address/subdenom"; // Replace with the actual denomination
  const msgs: EncodeObject[] = [
    {
      typeUrl: "/nibiru.tokenfactory.v1.MsgMint",
      value: MsgMint.fromPartial({
        sender: address,
        coin: { denom, amount: "1000" }, // Specify the denomination and amount
        mintTo: "receiver-address", // Address receiving minted tokens
      }),
    },
  ];

  const fee = "auto";
  const tx = await txClient.signAndBroadcast(address, msgs, fee);
  const result = MsgMintResponse.decode(tx.msgResponses[0].value);

  console.log("Mint Response:", result);
};

```

### Burn Tokens

Burn tokens by specifying the `denom` and amount to remove from circulation.

```javascript
import { EncodeObject } from "@cosmjs/proto-signing";
import { MsgBurn, MsgBurnResponse } from "@nibiruchain/nibijs/dist/src/protojs/nibiru/tokenfactory/v1/tx";

const burn = async () => {
  const denom = "tf/nibi1creator-address/subdenom"; // Replace with the actual denomination
  const msgs: EncodeObject[] = [
    {
      typeUrl: "/nibiru.tokenfactory.v1.MsgBurn",
      value: MsgBurn.fromPartial({
        sender: address,
        coin: { denom, amount: "1000" },
        burnFrom: "holder-address", // Address holding the tokens
      }),
    },
  ];

  const fee = "auto";
  const tx = await txClient.signAndBroadcast(address, msgs, fee);
  const result = MsgBurnResponse.decode(tx.msgResponses[0].value);

  console.log("Burn Response:", result);
};

```

### Change Token Administrator

Update the administrator of a specific denomination.

```javascript
import { EncodeObject } from "@cosmjs/proto-signing";
import { MsgChangeAdmin, MsgChangeAdminResponse } from "@nibiruchain/nibijs/dist/src/protojs/nibiru/tokenfactory/v1/tx";

const changeAdmin = () => {
  const signerAddress = address; // from cosmos
  const fee = "auto"; // You can specify the fee if needed
  const denom = "tf/nibi1creator-address/subdenom"; // Replace with the actual denomination
  const msgs: EncodeObject[] = [
    {
      typeUrl: "/nibiru.tokenfactory.v1.MsgBurn",
      value: MsgChangeAdmin.fromPartial({
        sender: address,
        denom,
        newAdmin: "", // new address
      }),
    },
  ]
  
  const tx = await txClient.signAndBroadcast(
    address, 
    msgs, 
    fee
  );

  const rtx = MsgChangeAdminResponse.decode(tx.msgResponses[0].value)
  console.log(rtx)
}
```

### Query Denomination Info

Retrieve details about a specific denomination.

```javascript
const denom = "tf/nibi1creator-address/subdenom"; // Replace with the actual denomination
const queryDenomInfo = await txClient.nibiruExtensions.query.tokenFactory.denomInfo({ denom });
console.log("Denomination Info:", queryDenomInfo);
```

### Query Denominations by Creator

List all denominations created by a specific wallet address.

```javascript
const creator = "nibi1creator-address"; // Replace with the creator's Bech32 address
const queryDenoms = await txClient.nibiruExtensions.query.tokenFactory.denoms({ creator });
console.log("Created Denominations:", queryDenoms);

```

### Query TokenFactory Module Parameters

Fetch the current parameters of the TokenFactory module.

```javascript
const queryParams = await txClient.nibiruExtensions.query.tokenFactory.params();
console.log("TokenFactory Parameters:", queryParams);
```

## Related Pages

- [NibiJS Getting Started](./getting-started.md)
- [NibiJS Connecting wiht a wallet extension](./connect-wallet.md)
- [Creating Fungible Tokens (Nibiru CLI)](./../../cli/tf.md)
- [Creating Fungible Tokens (Wasm Guides](./../../cw/tf.md)
