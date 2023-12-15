# CHAOSNET

- [How to use "chaosnet"](#how-to-use-chaosnet)
- [How to force pull images from the registry](#how-to-force-pull-images-from-the-registry)
- [Endpoints](#endpoints)
- [FAQ](#faq)
  - [`make chaosnet` says that "Additional property name is not allowed"](#make-chaosnet-says-that-additional-property-name-is-not-allowed)

## How to use "chaosnet" 

Before running

```sh
make chaosnet
```

you need to log into our private Docker image registry in order to pull the private images. Go to <https://github.com/settings/tokens/new> and generate a new token with `read:packages` scope. Copy the access token to your clipboard.

Next, run

```sh
docker login ghcr.io
```

 and enter your GitHub username for the `username` field, and your personal access token for the password.

Now you can run

```sh
make chaosnet
```

## How to force pull images from the registry

By default, images won't re-fetch from upstream registries. To force a pull, you can run

```sh
make chaosnet-build
```

to force re-build and pull images.

## Endpoints

- `http://localhost:5555` -> GraphQL server
- `http://localhost:26657` -> Tendermint RPC server
- `tcp://localhost:9090` -> Cosmos SDK gRPC server
- `http://localhost:1317` -> Cosmos SDK LCD (REST) server
- `http://localhost:8000` -> Faucet server (HTTP POST only)

## FAQ

### `make chaosnet` says that "Additional property name is not allowed"

Make sure to update your docker application to version 23.0.1
