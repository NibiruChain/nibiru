# contrib/docker-compose

- [contrib/docker-compose](#contribdocker-compose)
  - [docker-compose-chaosnet](#docker-compose-chaosnet)
  - [Usage](#usage)
    - [Single validator node + pricefeeder](#single-validator-node--pricefeeder)
    - [Two validator nodes + pricefeeder + IBC relayer](#two-validator-nodes--pricefeeder--ibc-relayer)
    - [Single validator node + heartmonitor](#single-validator-node--heartmonitor)
    - [Other Commands](#other-commands)
  - [Services Overview](#services-overview)
    - [Nibiru node services](#nibiru-node-services)
    - [Pricefeeder services](#pricefeeder-services)
    - [Hermes IBC relayer services](#hermes-ibc-relayer-services)
    - [Heart Monitor Services](#heart-monitor-services)
  - [Reference Materials](#reference-materials)

## docker-compose-chaosnet

The [docker-compose-chaosnet.yml](./docker-compose-chaosnet.yml) Compose
Specification configures Docker services to set up a fully functional, local
development environment tailored for Nibiru Chain, orchestrating several
Nibiru-specific containers.

Features:

- Different ports are utilized to mimic a multi-chain configuration on a single machine.
- Enables testing of cross-chain transactions, chain health monitoring, and more in a local Docker context across two chains.

## Usage

### Single validator node + pricefeeder

```sh
docker compose -f docker-compose-chaosnet.yml up
```

### Two validator nodes + pricefeeder + IBC relayer

```sh
docker compose -f docker-compose-chaosnet.yml --profile ibc up 
```

### Single validator node + heartmonitor

```sh
docker compose -f docker-compose-chaosnet.yml --profile heartmonitor up
```

### Other Commands

- `docker compose up`: Start the services.
- `docker compose down`: Stop the services.
- `docker compose restart`: Restart all services.
- `docker compose ps`: List containers, their status, ports, etc.
- `docker compose logs`: View std output from containers

## Services Overview

### Nibiru node services

- `nibiru-0` and `nibiru-1` (Service): Represents two distinct Nibiru Chain nodes (nibiru-0 and nibiru-1)
  running on different ports, using unique mnemonics and chain IDs, imitating two
  independent blockchain networks.

### Pricefeeder services

- `pricefeeder-0` and `pricefeeder-1` (Service): Two price feeder services that push price
  data to the respective Nibiru nodes.

### Hermes IBC relayer services

An IBC relayer is set up to connect the two chains using [hermes](https://hermes.informal.systems/).

1. `hermes-keys-task-0` and `hermes-keys-task-1` (Service): Tasks to generate
   keys for the validators on `nibiru-0` and `nibiru-1`.
2. `hermes-client-connection-channel-task` (Service): Creates a new channel
   between the two chains AND a client connection underlying this new channel.
3. `hermes` (Service): Runs and maintains an IBC relayer for the two chains.
   Relayers are off-chain processes responsible for reading data from one chain
   and submitting it to another. These relayers listen for IBC events on one
   chain, then construct and broadcast a corresponding transaction to the other
   chain. Relayers essentially submit packets between chains.

Brief IBC reference:

Put simply, **connections** represent a secure communication line between two
blockchain to transfer IBC **packets** (data). Once a connection is established,
light client of two chains, usually called the source chain and destination
chain, is established.

Once a connection is established, **channels** can be formed. A channel
represents a logical pathway for specific types communication over the connection
(like token transfers and other relaying of IBC packets.

### Heart Monitor Services

- `heartmonitor-db`: A postgres database for the heart monitor.

- `heartmonitor`: An indexing solution that populates a DB based on events and
  block responses emitted from Nibiru nodes.

- `graphql`: GraphQL API for the heart monitor data. Used in the Nibiru web app
  and other off-chain tools.

Repository: [NibiruChain/go-heartmonitor](https://github.com/NibiruChain/go-heartmonitor).

## Reference Materials

- [Docker Compose file](https://docs.docker.com/compose/compose-file/03-compose-file/)
- [Docker Compose Specification](https://github.com/compose-spec/compose-spec/blob/master/spec.md)
- [IBC-Go Docs](https://ibc.cosmos.network)
