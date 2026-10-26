# Chaosnet

- Status: active
- Code scope: file `just-chaos.just`, directory `contrib/docker-compose/`,
  script `contrib/scripts/chaosnet.sh`, and the determinism check
  (`contrib/scripts/determinism.sh`, `contrib/scripts/determinism/`,
  `contrib/scripts/multival-genesis.sh`, `contrib/scripts/multival-load.sh`)
- Source of truth: file `just-chaos.just` and Compose file
  `contrib/docker-compose/docker-compose-chaosnet.yml`

Chaosnet is the Docker Compose environment for multi-node Nibiru development.
It can run a primary validator, an optional second validator with Hermes IBC,
an optional Heart Monitor and GraphQL stack, and an optional network of four
validators on one chain (the `multival` profile).

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
| `just chaos up-multival` | Starts four validators on one chain and preserves existing named volumes. |
| `just chaos determinism` | Runs the app hash agreement check on a fresh `multival` network. |
| `just chaos determinism-selftest` | Self-test: plants a non-deterministic `EndBlock` and passes only if the check catches it. |
| `just chaos determinism-all` | Runs `determinism-selftest`, then `determinism`. Stops if the self-test fails. |
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

| `nibiru-val-0` with multival | `http://localhost:46657` | | EVM JSON-RPC `http://localhost:48545` |
| `nibiru-val-1` with multival | `http://localhost:46658` | | EVM JSON-RPC `http://localhost:48546` |
| `nibiru-val-2` with multival | `http://localhost:46659` | | EVM JSON-RPC `http://localhost:48547` |
| `nibiru-val-3` with multival | `http://localhost:46660` | | EVM JSON-RPC `http://localhost:48548` |

The Heart Monitor Postgres service publishes port `5433` on the host.

## Multi-validator network

The `multival` profile runs four `nibid` validators on chain
`nibiru-localnet-3`. Unlike `nibiru-0` and `nibiru-1`, which are separate
single-validator chains, these validators share one genesis and must agree
on every block.

- Image `nibiru-multival:local`, built from
  `contrib/docker-compose/Dockerfile.multival`. It holds only `nibid`, `jq`,
  and `curl`; genesis is created at container start, not at image build.
  `Dockerfile.multival.dockerignore` limits the build context to the Go
  sources that `nibid` needs.
- The one-shot service `multival-genesis` runs
  `contrib/scripts/multival-genesis.sh`. It creates a home per validator,
  funds the validators and the load accounts, collects one gentx per
  validator into a single `genesis.json`, and sets `persistent_peers` so
  each validator dials the other three. It also enables the EVM JSON-RPC
  server. Everything lives in the `multival-data` volume under `/net`.
- `nibiru-val-0` to `nibiru-val-3` start after `multival-genesis` exits.

Start it with `just chaos up-multival`. The profile keeps its volume across
`just chaos down`; `just chaos destroy` removes it.

## Determinism check

`just chaos determinism` runs `contrib/scripts/determinism.sh`. It catches
non-determinism in Nibiru's state machine that single-process tests miss,
such as a Go map iterated while writing state, a float in the EVM gas path,
or a Wasm call that reads wall-clock time. On mainnet these show up as
`wrong Block.Header.AppHash` on the validators that diverge.

The script:

1. Builds the image and starts the `multival` profile from a fresh volume.
2. Waits for height 3, then sends load to every node for `LOAD_SECONDS`
   (default 120), so different proposers build blocks with different
   transaction orders:
   - EVM: `contrib/scripts/determinism evm-load` signs contract deploys,
     contract calls, and value transfers and sends them to each node's
     JSON-RPC with `eth_sendRawTransaction`.
   - Bank and Wasm: `contrib/scripts/multival-load.sh` runs in a one-off
     container and uses `nibid tx bank send`, `nibid tx wasm store`,
     `instantiate`, and `execute` against each node. The contract is
     `evm/precompile/test/hello_world_counter.wasm`.
3. Every 10 seconds of the load, kills one validator with `SIGKILL` and
   starts it again, or restarts it, rotating through the four. Nodes replay
   blocks from disk on startup.
4. Waits for every node to reach the last height plus `EXTRA_HEIGHTS`
   (default 10). `determinism compare` then reads `/block` and
   `/block_results` for every height from every node and compares
   `block_id.hash`, `header.app_hash`, `header.last_results_hash`, and each
   transaction's `code` and `gas_used`.
5. Removes the network, unless `KEEP_NETWORK=true`.

It exits non-zero when:

- nodes disagree on any compared field; it prints the first height and
  field;
- a node's height does not move for 60 seconds before the target. It
  prints each node's height and app hash from `/abci_info`, the first
  height where app hashes differ, and the last 100 log lines of each
  stopped node.

A single diverging validator stops and the other three continue, so the
report compares its app hash with the next block's header on a healthy
node. When every node diverges the chain halts, and the report groups the
nodes by app hash at the height where they stopped.

One run takes about 3 minutes on a 12-core machine with a built image: 15
seconds to start, 2 minutes of load, and about 25 seconds to reach the
target height and compare about 90 heights. Building the image from scratch adds about 2 minutes.

| Variable | Default | Meaning |
| --- | --- | --- |
| `LOAD_SECONDS` | `120` | Load and perturbation time. |
| `EXTRA_HEIGHTS` | `10` | Heights past the end of load to wait for. |
| `MULTIVAL_IMAGE` | `nibiru-multival:local` | Image tag. |
| `MULTIVAL_BUILD_TAGS` | none | Extra Go build tags for `nibid`. |
| `KEEP_NETWORK` | `false` | Leave the network running after the check. |

### Planted bug

`just chaos determinism-selftest` builds `nibiru-multival:fault-injection`
with the test-only build tag `nibiru_fault_injection`. With that tag,
`evm/evmstate/fault_injection.go` writes each node's
`time.Now().UnixNano()` to the EVM store in `EndBlock`, so every validator
commits a different app hash after height 1. The recipe passes only if the
check fails and names the first height where app hashes differ. An
unrelated failure, such as a crash or a build error, does not count as a
pass. Never ship a binary built with that tag.

The self-test is a regression test for the detector. `just chaos
determinism-all` runs it first and stops if it fails, because a clean
`determinism` run means nothing if the check cannot see a planted bug. It
then runs `determinism`.

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
