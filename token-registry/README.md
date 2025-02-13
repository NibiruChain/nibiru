# Nibiru/token-registry

This directory implements the Nibiru Token Registry by providing a means to
register offchain digital token metadata to onchain identifiers for use with
applications like wallets.

## Exporting for the cosmos/chain-registry

The command to generate files from the Nibiru Token Registry is
```bash
just gen-token-registry
```
   
If you don't have the `just` command,
```bash
cargo install just
```

## erc20s_official.json

This file maintains a registry of known ERC20 tokens. It's intended for use in the [Nibiru web
application](https://app.nibiru.fi) and the Nibiru Indexer that provides data on
tokens and token balances.

### Fields of the `erc20s_official.json` configuration file
- `contract_addr`: ERC20 smart contract address of the token
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

Related Ticket: https://github.com/NibiruChain/go-heartmonitor/issues/378
