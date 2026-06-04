---
order: 0
description: Guide to running full nodes and validator full nodes on the Nibiru blockchain.
---

# Nibiru Nodes

{{ $frontmatter.description }}

All of nodes that make up Nibiru run the [same code/binary (Nibiru)](https://github.com/NibiruChain/nibiru/releases), but their configuration and purpose determine their role in the network.

## Mainnet Nodes

| Section | Synopsis |
| --- | --- |
| [Run a Full Node (Mainnet)](./full-nodes/full-node-mainnet.md) | Guide to running a Nibiru mainnet full node: hardware requirements, installation options, sync and upgrade workflows, chain initialization, and memory optimization. |
| [Become a Validator (Mainnet)](./validators/validator-mainnet.md) | Instructions for running a validator node for Nibiru Mainnet. |
| [Set up a Price Feeder (Mainnet)](./validators/pricefeeder-mainnet.md) | Instructions for validators to set up a Mainnet price feeder. |


## Node Types

| Node Type | Description |
| --- | --- |
| **Validator Node** | Validators are responsible for producing blocks and participating in consensus (via NibiruBFT). They secure the chain by signing blocks, relaying transactions, and maintaining uptime. Validators must stake NIBI and are subject to rewards and slashing.                |
| **Full Node**      | Full nodes maintain the full state of the blockchain and verify all blocks, but they do not participate in consensus. They are essential for decentralization, as they provide transaction relaying, independent verification, and serve as sentries for validator setups. |
| **Sentry Node**    | Sentry nodes are a type of full node configured to sit between a validator and the public network. They protect validators from DDoS attacks by acting as a shield layer.                                                                                                  |
| **RPC Node**       | RPC nodes expose APIs (both REST and gRPC) for developers, exchanges, wallets, and explorers to query chain data or broadcast transactions. They are often tuned for high availability and throughput.                                                                     |
| **Archival Node**  | Archival nodes keep the full blockchain history, including all historical states and blocks. This makes them valuable for block explorers, research, analytics, or applications that need historical queries beyond the pruning settings of standard nodes.                |

All Nibiru nodes run the same binary, but the role comes from how you configure and operate them (consensus participation, API service, pruning settings, etc.).All Nibiru nodes run the same binary, but the role comes from how you configure and operate them (consensus participation, API service, pruning settings, etc.).

### Helpful Info for Validators

| Section | Synopsis |
| --- | --- |
| [Binary Upgrades](./validators/upgrades.md) | Chain upgrade procedures: how validators swap binaries at upgrade block heights. |
| [Set up a Price Feeder (Testnet)](./validators/pricefeeder-testnet.md) | Instructions for validators to set up a Testnet price feeder. |
| [Reset a Validator Node (Testnet)](./validators/node-reset.md) | Instructions for validators to rebuild in the case of a Testnet chain reset. |

### Reference

| Section | Synopsis |
| --- | --- |
| [Node Daemon](./full-nodes/node-daemon.md) | The main endpoint of an SDK application: the daemon client, otherwise known as the full-node client. |
| [Systemctl and Services](./full-nodes/systemctl.md) | System and service manager for Linux operating systems, designed for the management and configuration of services. |
| [Cosmovisor](./full-nodes/cosmovisor.md) | Process manager for Cosmos-SDK application binaries that monitors the governance module for incoming chain upgrade proposals. |


## Testnet Nodes

| Section | Synopsis |
| --- | --- |
| [Run a Full Node (Testnet)](./full-nodes/index.md) | Run a full node on Nibiru testnet: setup, sync options, and network joining procedures. |
| [Become a Validator (Testnet)](./validators/index.md) | Instructions for running a validator node for Nibiru Testnet. |


