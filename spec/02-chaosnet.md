# Chaosnet

- Status: active
- Code scope: file `just-chaos.just`, directory `contrib/docker-compose/`, and
  script `contrib/scripts/chaosnet.sh`
- Source of truth: file `just-chaos.just` and Compose file
  `contrib/docker-compose/docker-compose-chaosnet.yml`

Chaosnet is the Docker Compose environment for multi-node Nibiru development.
It can run a primary validator, an optional second validator with Hermes IBC,
and an optional Heart Monitor and GraphQL stack.

Use the `just chaos` command group from the repository root.

## Prerequisites

Install Docker Engine with Docker Compose V2 and command `just`. Authenticate
to `ghcr.io` when using the Heart Monitor profile if the local Docker client
cannot pull its image.

## Start and stop

From the repository root, use these commands:

| Command | Result |
| --- | --- |
| `just chaos up` | Starts one validator and preserves existing named volumes. |
| `just chaos up-ibc` | Starts two validators and Hermes, and preserves existing named volumes. |
| `just chaos up-hm` | Starts one validator with Postgres, Heart Monitor, and GraphQL. |
| `just chaos build` | Rebuilds Chaosnet images without cache and pulls base images. |
| `just chaos down` | Stops all profiles and preserves named volumes. |
| `just chaos destroy` | Stops all profiles and removes named volumes after confirmation. |

Command `just chaos destroy` deletes local Chaosnet data. Use command `just
chaos down` when you need to retain the current local chain state.

## Logs and shells

```bash
just chaos logs
just chaos logs-hm
just chaos sh-nibiru-0
just chaos sh-nibiru-1
just chaos sh-go-hm
```

The `nibiru-1` shell requires the IBC profile. The `sh-go-hm` command opens a
shell in the `heartmonitor` service and requires the Heart Monitor profile.

## Endpoints

| Service | RPC | gRPC | API or other endpoint |
| --- | --- | --- | --- |
| `nibiru-0` | `http://localhost:26657` | `localhost:9090` | `http://localhost:1317` |
| `nibiru-1` with IBC | `http://localhost:36657` | `localhost:19090` | `http://localhost:11317` |
| GraphQL with Heart Monitor | | | `http://localhost:5555` |
| Hermes with IBC | | | `http://localhost:3000`, `http://localhost:3001` |

The Heart Monitor Postgres service publishes port `5433` on the host.

## IBC walkthrough

The following commands assume command `just chaos up-ibc` is running.

Enter the primary validator container:

```bash
just chaos sh-nibiru-0
```

Send a transfer to the second validator:

```bash
nibid tx ibc-transfer transfer transfer \
  channel-0 \
  nibi18mxturdh0mjw032c3zslgkw63cukkl4q5skk8g \
  1000000unibi \
  --from validator \
  --fees 5000unibi \
  --yes | jq
```

In another shell, enter the second validator and query its validator balance:

```bash
just chaos sh-nibiru-1
nibid config node "http://localhost:36657"
nibid q bank balances "$(nibid keys show validator -a)" | jq
```

### Interchain accounts

Use controller chain `nibiru-0` to register an interchain account on host chain
`nibiru-1`:

```bash
FROM=nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl

nibid tx interchain-accounts controller register \
  connection-0 \
  --from "$FROM" \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.025unibi \
  --yes
```

Query the registered address:

```bash
nibid q interchain-accounts controller interchain-account \
  nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl \
  connection-0 | jq
```

Fund the returned address from `nibiru-1`, then verify its balance:

```bash
nibid tx bank send \
  nibi18mxturdh0mjw032c3zslgkw63cukkl4q5skk8g \
  <ica-address> \
  1000000unibi \
  --from validator \
  --fees 5000unibi \
  --yes | jq

nibid q bank balances <ica-address> | jq
```

On `nibiru-0`, generate and send an interchain-account packet:

```bash
nibid tx interchain-accounts host generate-packet-data \
  "<msg-json>" --encoding proto3 | jq
nibid tx interchain-accounts controller send-tx connection-0 packet.json \
  --from "$FROM" --yes | jq
```

Verify the resulting delegation on `nibiru-1`:

```bash
nibid q staking delegations <ica-address> | jq
```

## Troubleshooting

If Compose reports an unsupported property, update Docker Engine and Docker
Compose. If startup stalls, inspect the service logs with command
`just chaos logs`. The initial build pulls images, builds the local node
image, and the IBC profile also creates clients, a connection, and a channel.
