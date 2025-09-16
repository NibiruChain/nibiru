---
order: 3
---
# Nibid Keys & Faucet

To deploy a CosmWasm contract on Nibiru, set up a wallet with
nibid keys add wallet and acquire funds from the Nibiru Faucet
using Leap or Keplr. Remember to keep your mnemonic or private
key secure.{synopsis}

## Nibid Keys

First setup a wallet with the command below.

```bash
# add wallets for testing
nibid keys add wallet
```

To check which wallet is currently setup, run

```bash
nibid keys show -a wallet
```

## Faucet

Currently, only way to acquire funds for Nibiru's Testnets is via the [app.nibiru.fi/faucet](https://app.nibiru.fi/faucet).
First connect your wallet to the recommends [Wallet extensions](../../ecosystem/wallets/index.md), Leap or Keplr.
Then you should be able to request funds. You are limited to once per day.

In order to verify that your funds have been added to your account, you can query your balance by running:

```bash
nibid query bank balances $(nibid keys show -a wallet)
```

## Related Pages

- [Wallets](../../wallets/index.md)
- [Create a Nibiru Wallet Address](../../wallets/create-addr.md)
