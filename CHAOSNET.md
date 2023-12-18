# Chaosnet

- [Chaosnet](#chaosnet)
  - [What is Chaosnet?](#what-is-chaosnet)
  - [How to run "chaosnet"](#how-to-run-chaosnet)
  - [How to force pull images from the registry](#how-to-force-pull-images-from-the-registry)
  - [Endpoints](#endpoints)
  - [FAQ](#faq)
    - [`make chaosnet` says that "Additional property name is not allowed"](#make-chaosnet-says-that-additional-property-name-is-not-allowed)
    - [Does data persist between runs?](#does-data-persist-between-runs)

## What is Chaosnet?

Chaosnet is an expanded version of localnet that runs:

- two validators (nibiru-0 and nibiru-1)
- pricefeeders for each validator
- a hermes relayer between the two validators
- a faucet
- a postgres:14 database
- a heartmonitor instance
- a liquidator instance
- a graphql server

## How to run "chaosnet"

1. Make sure you have [Docker](https://docs.docker.com/engine/install/) installed and running
2. Make sure you have `make` installed
3. Docker login to ghcr.io

```bash
docker login ghcr.io
```

Enter your GitHub username for the `username` field, and your personal access token for the password.

4. Run `make chaosnet`

## How to force pull images from the registry

By default, most images (heart-monitor, liquidator, etc.) are cached locally and won't re-fetch from upstream registries. To force a pull, you can run

```sh
make chaosnet-build
```

## Endpoints

- `http://localhost:5555` -> GraphQL server
- `http://localhost:26657` -> Tendermint RPC server
- `tcp://localhost:9090` -> Cosmos SDK gRPC server
- `http://localhost:1317` -> Cosmos SDK LCD (REST) server
- `http://localhost:8000` -> Faucet server (HTTP POST only)

## FAQ

### `make chaosnet` says that "Additional property name is not allowed"

Make sure to update your docker application to version >=23.0.1

### Does data persist between runs?

No, all volumes are deleted and recreated every time you run `make chaosnet`. This is to ensure that you always start with a clean network.
