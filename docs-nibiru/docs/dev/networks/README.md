---
order: 0
---

# Nibiru Networks and RPCs

Nibiru is as a distributed, peer-to-peer network. In order to engage with this
network, whether to query the state of it from or to broadcast
transactions, one must interface through the endpoint of a full node that is
connected of the network. {synopsis}

Table of Contents:

- [Mainnet (Real Funds)](#mainnet-real-funds)
- [Permanent Testnet](#permanent-testnet)
- [Localnet: Local Test Network](#localnet-local-test-network)
- [Related Pages](#related-pages)

## Mainnet (Real Funds)

Nibiru's mainnet network, Cataclysm-1, is the blockchain where real economic activities take place.

| Blockchain Network | Nibiru (Mainnet) |
| --- | --- |
| EVM RPC | https://evm-rpc.nibiru.fi |
| EIP-155 Chain ID | 6900 |
| EIP-155 Chain ID (Hex) | 0x1AF4 |
| CSDK Chain-ID | cataclysm-1 |
| CometBFT RPC | https://rpc.nibiru.fi:443 |

<!-- TODO: NibiJS Mainnet -->

Nibiru CLI Config: Mainnet

```bash
RPC_URL="https://rpc.nibiru.fi:443"
nibid config node $RPC_URL
nibid config chain-id cataclysm-1
nibid config broadcast-mode sync
nibid config # Prints your new config to verify correctness
```

## Permanent Testnet

Nibiru testnets are public networks that upgraded in advance of Nibiru's mainnet
as beta-testing environments.

Tokens on the testnet do not hold real monetary value. Please be careful not to
bridge or IBC transfer real tokens to testnet by mistake.

| Blockchain Network | Nibiru Testnet |
| --- | --- |
| EVM RPC | https://evm-rpc.testnet-2.nibiru.fi |
| EIP-155 Chain ID | 6911 |
| EIP-155 Chain ID (Hex) | 0x1AFF |
| CSDK Chain-ID | nibiru-testnet-2 |
| CometBFT RPC | https://rpc.testnet-2.nibiru.fi:443 |

NibiJS: Testnet

```js
import { Testnet, NibiruQuerier } from "@nibiruchain/nibijs"
const chain = Testnet(2) // corresponds to "nibiru-testnet-2"
const queryClient = await NibiruQuerier.connect(chain.endptTm)
```

Nibiru CLI Config: Testnet

```bash
RPC_URL="https://rpc.testnet-1.nibiru.fi:443"
nibid config node $RPC_URL
nibid config chain-id nibiru-testnet-1
nibid config broadcast-mode sync
nibid config # Prints your new config to verify correctness
```

## Localnet: Local Test Network

| Blockchain Network | Nibiru Localnet |
| --- | --- |
| EVM RPC | http://127.0.0.1:8545 |
| EIP-155 Chain ID | 6930 |
| EIP-155 Chain ID (Hex) | 0x1B0A |
| CSDK Chain-ID | nibiru-localnet-0 |
| CometBFT RPC | http://localhost:26657 |

"Localnets" are a local instances of the Nibiru network. A local environment is
no different from a real one, except that it has a single validator running on
your host machine. Localnet is primarily used as a controllable, isolated
development environment for testing purposes.

Similar to testnet, tokens on Localnet do not hold real monetary value. Please be
careful not to bridge or IBC transfer real tokens to localnet by mistake.

NibiJS: Localnet

```js
import { Localnet, NibiruQuerier } from "@nibiruchain/nibijs"
const chain = Localnet
const queryClient = await NibiruQuerier.connect(chain.endptTm)
```

Nibiru CLI Config: Localnet

```bash
RPC_URL="http://localhost:26657"
nibid config node $RPC_URL
nibid config chain-id nibiru-localnet-0
nibid config broadcast-mode sync
nibid config # Prints your new config to verify correctness
```

## EVM Network Configs

| Display Name     | Chain Namespace | Chain ID | RPC Target                             |
|------------------|-----------------|----------|----------------------------------------|
| Nibiru Mainnet   | 420             | 0x1AF4   | `https://evm-rpc.nibiru.fi/`           |
| Nibiru Testnet 1 | 500             | 0x1C2A   | `https://evm-rpc.testnet-1.nibiru.fi/` |
| Nibiru Devnet 1  | 500             | 0x1C34   | `https://evm-rpc.devnet-1.nibiru.fi/`  |
| Nibiru Devnet 3  | 500             | 0x1C36   | `https://evm-rpc.devnet-3.nibiru.fi/`  |

## Related Pages

- [Developer Hub: Build on Nibiru](../)
