# Chaosnet

Chaosnet is a Docker-based local multi-service environment for Nibiru development.

It can run:

- one or two validator nodes (`nibiru-0`, `nibiru-1`)
- Hermes IBC relayer services (with `ibc` profile)
- Heart Monitor stack (Postgres, indexer, GraphQL with `heartmonitor` profile)

## Prerequisites

1. Install and start [Docker](https://docs.docker.com/engine/install/).
2. Install command `make`.
3. (Optional) Authenticate to `ghcr.io` if you use the `heartmonitor` profile:

```bash
docker login ghcr.io
```

## Start Chaosnet

From the repository root:

```bash
make chaosnet
```

Other profiles:

- command `make chaosnet-ibc`: starts two validators plus Hermes relayer
- command `make chaosnet-heartmonitor`: starts validator plus Heart Monitor stack

To force rebuilding images:

```bash
make chaosnet-build
```

To stop and clean volumes:

```bash
make chaosnet-down
```

## Helpful Commands

```bash
make chaosnet-logs
make chaosnet-ssh-nibiru-0
make chaosnet-ssh-nibiru-1
```

These Make targets are defined in file `contrib/make/chaosnet.mk`.

## Endpoints

### Core Nibiru Node (`nibiru-0`)

- `http://localhost:26657` - Tendermint RPC
- `tcp://localhost:9090` - Cosmos SDK gRPC
- `http://localhost:1317` - Cosmos SDK LCD (REST)

### Secondary Node (`nibiru-1`, `ibc` profile)

- `http://localhost:36657` - Tendermint RPC
- `tcp://localhost:19090` - Cosmos SDK gRPC
- `http://localhost:11317` - Cosmos SDK LCD (REST)

### Other Services

- `http://localhost:5555` - Heart Monitor GraphQL (`heartmonitor` profile)
- `http://localhost:3000` and `http://localhost:3001` - Hermes ports (`ibc` profile)

## IBC Walkthrough

The following examples assume `make chaosnet-ibc` is running.

### Interchain Transfers

1. Enter `nibiru-0`:

```sh
docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it nibiru-0 /bin/sh
```

2. Send an IBC transfer to `nibiru-1`:

```sh
nibid tx ibc-transfer transfer transfer \
  channel-0 \
  nibi18mxturdh0mjw032c3zslgkw63cukkl4q5skk8g \
  1000000unibi \
  --from validator \
  --fees 5000unibi \
  --yes | jq
```

3. In another shell, enter `nibiru-1` and query balances:

```sh
docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it nibiru-1 /bin/sh
nibid config node "http://localhost:36657"
nibid q bank balances "$(nibid keys show validator -a)" | jq
```

### Interchain Accounts

Goal: use controller chain `nibiru-0` to drive an ICA on host chain `nibiru-1`.

1. On `nibiru-0`, register an ICA:

```sh
FROM=nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl

nibid tx interchain-accounts controller register \
  connection-0 \
  --from "$FROM" \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.025unibi \
  --yes
```

2. Query the ICA address:

```sh
nibid q interchain-accounts controller interchain-account \
  nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl \
  connection-0 | jq
```

3. Fund the ICA from `nibiru-1` and verify:

```sh
nibid tx bank send \
  nibi18mxturdh0mjw032c3zslgkw63cukkl4q5skk8g \
  <ica-address> \
  1000000unibi \
  --yes | jq

nibid q bank balances <ica-address> | jq
```

4. Create and send an ICA packet from `nibiru-0` to delegate on `nibiru-1`:

```sh
nibid tx interchain-accounts host generate-packet-data "<msg-json>" --encoding proto3 | jq
nibid tx interchain-accounts controller send-tx connection-0 packet.json --from "$FROM" --yes | jq
```

5. Verify delegation on `nibiru-1`:

```sh
nibid q staking delegations <ica-address> | jq
```

## FAQ

### `make chaosnet` Says "Additional Property Name Is Not Allowed"

Use Docker version `23.0.1` or newer.

### Does Data Persist Between Runs?

No. Command `make chaosnet-down` removes Chaosnet volumes so each fresh start begins from a clean state.

### `make chaosnet` Takes a Long Time

The first run builds/pulls images and creates IBC channels. Check logs with command `make chaosnet-logs` if startup stalls.
