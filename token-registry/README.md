# Nibiru/token-registry

This directory implements the Nibiru Token Registry by providing a means to
register offchain digital token metadata to onchain identifiers for use with
applications like wallets.

<img src="https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/nibiru-web-app.png">

## Generation Script

The command to generate files from the Nibiru Token Registry is
```bash
just gen-token-registry
```
   
If you don't have the `just` command, run this first.
```bash
cargo install just
```

This produces several files:
- `dist/cosmos-assetlist.json`: An external JSON file for use inside the
[cosmos/chain-registry GitHub repo](https://github.com/cosmos/chain-registry/tree/master/nibiru).
- `token-registry/official_erc20s.json`: Verified ERC20s for the Nibiru web app
- `token-registry/official_bank_coins.json`: Bank Coins in the Nibiru web app

## official_erc20s.json

This file maintains a registry of known ERC20 tokens. It's intended for use in the [Nibiru web
application](https://app.nibiru.fi) and the Nibiru Indexer that provides data on
tokens and token balances.

### Fields of the `official_erc20s.json` configuration file
- `contractAddr`: ERC20 smart contract address of the token
- `displayName`: Concise display name of the token. Example: "Wrapped Ether".
- `symbol`: Symbol, or ticker, of the digital asset.  
  Example: "ETH", "NIBI"
- `decimals`: The number of decimal places, or power of 10, that is used for the
  display unit of the token. Said more concretely, the smallest unit of
  the token, "base unit", is such that 1 TOKEN == 10^{decimals} base units.  
   - Example: NIBI has 6 decimals. This means the base unit seen onchain is 10^{-6} NIBI.
   - Example: ETH has 18 decimals. This means the base unit of ETH is 10^{-18} ether.
- `logoSrc`: GitHub static asset link for the logo image that will often appear
inside a circular frame.  
  Example: "raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/0000_nibiru.png"
- `priceInfo`: (Optional) 

Related Ticket: https://github.com/NibiruChain/go-heartmonitor/issues/378

## Exporting for the cosmos/chain-registry

1. Run the generation script (`just gen-token-registry`) 
2. Copy or move the "dist/cosmos-assetlist.json" file to replace
   "nibiru/assetlist.json" in the "cosmos/chain-registry" repo.
