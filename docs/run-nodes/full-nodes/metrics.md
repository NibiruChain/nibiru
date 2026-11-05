# Node metrics

SDK telemetry exposes application activity alongside CometBFT consensus metrics.
Enable both Prometheus exporters to scrape them together at
`http://localhost:26660/metrics`. They use the same registry inside `nibid`, so
SDK metrics do not need an extra server, sidecar, or scrape target.

In `app.toml`, configure:

```toml
[telemetry]
enabled = true
service-name = "nibiru"
prometheus-retention-time = 600
global-labels = [["chain_id", "cataclysm-1"]]
```

In `config.toml`, configure:

```toml
[instrumentation]
prometheus = true
prometheus_listen_addr = ":26660"
```

Restart the node after changing its configuration. Keep the metrics listener on
a private network. Telemetry remains disabled by default.

The API server also exposes `/metrics?format=prometheus` when API and telemetry
are enabled. Plain API `/metrics` returns the in-memory JSON representation.
The CometBFT listener on port `26660` returns Prometheus text without parameters
and works even when the API server is disabled.

## Application metrics

| Metric | Labels | Meaning |
| --- | --- | --- |
| `nibiru_sdk_tx_exec_duration` | `result` | Histogram of application transaction execution duration in seconds. |
| `nibiru_sdk_tx_msgs_total` | `message_type`, `tx_result` | Top-level message occurrences in decoded delivered transactions. |
| `nibiru_sdk_query_duration` | `route`, `result` | Histogram of ABCI application query handling duration in seconds. |

The histograms expose `_bucket`, `_sum`, and `_count` series. Their `HELP` text
states the seconds unit. Set Grafana duration panels to seconds. Outcomes are
`successful` or `failed`, and global labels apply to all three metrics. The
`service-name` setting prefixes legacy SDK metrics, but these three names are
fixed. Histogram buckets and message counters remain cumulative for the life of
the process; the retention setting governs expiry of inactive legacy metrics.

Transaction timing covers `runTx`, including decoding, ante handling, message
execution, post handling, recovery, and gas accounting. It excludes mempool
waiting, networking, block commit, and streaming listeners. Only delivery emits
transaction and message metrics. CheckTx, rechecks, simulation, and proposal
processing do not contribute.

Each top-level message is counted, including repeated types in one transaction.
`tx_result` is the final outcome of the containing transaction. A failed
transaction still contributes its decoded messages, even when ante handling
rejects it or a later message fails. These are included-message counts, not
counts of successful message execution. Malformed transactions that cannot be
decoded have no message counts. Nested Authz messages and Wasm-dispatched SDK
messages are not counted separately.

Query timing covers the ABCI application handler. It does not measure complete
HTTP, EVM JSON-RPC, or direct gRPC requests. Registered gRPC query methods keep
their full route names. Other ABCI paths use `/app`, `/store`, `/p2p`, or `unknown`.
Legacy query metrics use the same grouping to bound metric cardinality.

Every synced node executes the same blockchain transactions. Select one pod for
chain transaction and message rates. Summing counts across replicas duplicates
chain activity. Compare execution latency per pod when investigating slow nodes.

## PromQL examples

Transaction rate from one pod:

```promql
sum(rate(nibiru_sdk_tx_exec_duration_count{pod="$pod"}[5m]))
```

Transaction p95 duration in seconds:

```promql
histogram_quantile(0.95, sum by (le) (
  rate(nibiru_sdk_tx_exec_duration_bucket{pod="$pod"}[5m])
))
```

Message rate grouped by message type and transaction outcome:

```promql
sum by (message_type, tx_result) (
  rate(nibiru_sdk_tx_msgs_total{pod="$pod"}[5m])
)
```

Legacy SDK timers such as `nibiru_store_iavl_commit` and
`nibiru_begin_blocker` use milliseconds. Their summaries have a short quantile
window, so use `_sum` and `_count` rates for longer-window mean durations and
divide milliseconds by `1000` before displaying seconds. Do not aggregate
summary quantiles across pods. Legacy transaction gas gauges show the last
observed transaction, not a gas distribution or a block total.

CometBFT metrics add consensus rounds, step timing, block intervals, and ABCI
connection timing. Block intervals describe production cadence, not an
end-to-end measurement of propagation to every peer. Distributed tracing is a
separate follow-up tracked in [#658](https://github.com/NibiruChain/nibiru/issues/658).
