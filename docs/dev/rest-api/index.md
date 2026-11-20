# REST API (gRPC Gateway) Reference

Use Nibiru's REST API to query blockchain data and broadcast transactions from web applications.

The REST API provides HTTP endpoints that correspond to Nibiru's gRPC services. This allows you to interact with the blockchain using standard HTTP requests, making it accessible for web applications that don't support gRPC's HTTP/2 protocol.

The following reference documentation covers all available REST endpoints for Nibiru's blockchain modules.  

| REST Service | Summary |
| --- | --- |
| [nibiru/evm](./nibiru-evm.md) | EVM-compatible execution layer queries: account balances, contract bytecode, transaction receipts, and gas fees |
| [nibiru/devgas](./nibiru-devgas.md) | Retired developer fee sharing module: read-only queries for the fee shares left in state |
| [nibiru/tokenfactory](./nibiru-tokenfactory.md) | Custom token creation and management: query token metadata, admin permissions, and creator-specific denominations |
| [nibiru/sudo](./nibiru-sudo.md) | Administrative permissions: query authorized sudo accounts for privileged blockchain operations |
| [nibiru/epochs](./nibiru-epochs.md) | Time-based triggers: query current epoch information and epoch-based module scheduling |
| [nibiru/inflation](./nibiru-inflation.md) | Token economics: query circulating supply, mint provisions, and inflation parameters |

Example:
```bash
curl -X GET /nibiru/evm/v1/eth_account/{address} \
  -H 'Accept: application/json'
```
