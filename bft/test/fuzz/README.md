# fuzz

Native Go fuzz tests (`go test -fuzz`) for the Nibiru BFT code paths that
decode bytes from peers and RPC clients.

| Target | Fuzz function | What it feeds |
| --- | --- | --- |
| `mempool-v0` | `FuzzMempoolV0` | `CheckTx` against the kvstore app |
| `mempool-v1` | `FuzzMempoolV1` | same, priority mempool |
| `p2p-addrbook` | `FuzzP2PAddrBook` | `AddrBook.AddAddress` with a JSON `NetAddress` |
| `p2p-pex` | `FuzzP2PPex` | `pex.Reactor.ReceiveEnvelope` with a decoded PEX message |
| `p2p-secret-connection` | `FuzzP2PSecretConnection` | `SecretConnection` `Write` then `Read` |
| `rpc-jsonrpc-server` | `FuzzRPCJSONRPCServer` | the HTTP JSON-RPC handler |
| `pubsub-query` | `FuzzPubsubQuery` | the event query parser in `libs/pubsub/query` |

## Running

From the repository root:

```sh
just bft fuzz-targets          # list target names
just bft fuzz p2p-pex 30s      # fuzz one target for 30 seconds
just bft test ./test/fuzz/...  # replay the seeds as unit tests
```

## Seeds and failing inputs

Each package keeps its seed corpus in `testdata/fuzz/<FuzzFunc>/`. Plain
`go test` runs every file there, so `just bft test` replays the seeds on every
PR.

When fuzzing finds a failing input, Go writes it to the same directory and
prints the command to re-run it. Commit that file with the fix so it stays a
regression test. The nightly workflow `.github/workflows/test-bft-fuzz.yml`
runs each target for 10 minutes and uploads any failing input as an artifact.

## OSS-Fuzz

`oss-fuzz-build.sh` builds the same tests for OSS-Fuzz with
`compile_native_go_fuzzer`. Nibiru is not enrolled in OSS-Fuzz.
