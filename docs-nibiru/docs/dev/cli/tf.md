---
order: 3
footer:
  newsletter: false
---

# Creating Fungible Tokens (Nibiru CLI)

The TokenFactory (tf) module on Nibiru Chain enables any account to create a new token with a specific format that includes the creator's address and a subdenomination. This method enables token creation without requiring permission and prevents problems related to name conflicts.

::: tip
See ["Nibid Setup"](./nibid-binary.md) for instructions on installing `nibid`.
:::

By using this module, a single account can create multiple denominations by providing a unique subdenomination for each one. Once a denomination is created, the original creator gains "admin" privileges over the token

Using the Nibiru CLI, users can create denominations mint, burn, and change admins for created tokens.

## Understanding the denomination format

The Nibiru Token Factory denomination format is in `tf/{creator-address}/{subdenom}`. Tokens created through the x/tokenfactory module are native to our chain and are prefixed with "tf," similar to the "ibc" prefix used for IBC assets. This prefixing convention helps avoid conflicts in naming.

In the `tokenfactory/.../state.proto` file, the TFDenom message defines a token factory (TF) denomination. Its standard representation is  `tf/{creator-address}/{subdenom}` where `{creator-address}` is the Bech32 address of the creator of the denomination, and `{subdenom}` is a unique suffix specific to the creator, used during a token factory "Mint" operation.

## Creating a Denomination

Before running, ensure that you have configured the correct chain as explained in [Nibiru Networks](../networks/README.md).

```bash
nibid tx tf create-denom <subdenom> \ 
  --from <creator-address> \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.025unibi \
  -y | jq
```

To verify denoms are created correctly, run a query on the creator address and ensure that the new subdenom is listed:

```bash
nibid query tf denoms <creator-address> | jq
```

## Minting & Burning Tokens

To mint, esure that you have the correct denomaination format setup:

```bash
COIN="<amount>tf/<creator-address>/<denomination>"

nibid tx tf mint $COIN \ 
  --from <creator-address> \
  --mint-to <receipient> \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.025unibi \
  -y | jq
```

To burn token, you also need the same denomaination format:

```bash
COIN="<amount>tf/<creator-address>/<denomination>"

nibid tx tf burn $COIN \
  --from <creator-address> \
  --burn-from <address-to-burn-from> \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.025unibi \
  -y | jq
```

To verify, you can check the receipient address's balance:

```bash
nibid query bank balances <address> | jq
```

## Change Token Admin

```bash
COIN="<amount>tf/<creator-address>/<denomination>"

nibid tx tf change-admin $COIN \ 
  <new-admin-address> \
  --from <creator-address> \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.025unibi \
  -y | jq
```

To verify, you can query the denom information:

```bash
COIN="tf/<creator-address>/<denomination>"

nibid query tf denom-info $COIN | jq
```
