# Nibiru Upgrade Module

This package preserves the native Cosmos upgrade messages and queries while
using the `x-upgrade` Rust contract as canonical storage and scheduling logic.
Shared protobuf types remain under `lib/cosmos-sdk/x/upgrade/types`.

See [the implementation specification](../../spec/08-contract-owned-upgrades.md)
and [contract build instructions](../../wasm-contracts/x-upgrade/README.md).

The enabling v2.22.0 handler runs ordinary module migrations, imports legacy
state, verifies the snapshot, removes only the replaced internal keys, and
activates the contract last. Outer `ApplyUpgrade` then records returned module
versions, increments protocol version, clears the plan, and records completion
through the contract-backed keeper. No EndBlock cutover is needed.

Historical contexts without the activation marker use legacy storage. IBC
proof-visible paths remain native. The same legacy backend supports SDK/IBC
test applications that do not include Nibiru Wasm.

## Native Interface

Existing height plans and governance cancellation retain their message types.
Time-target requests use the existing `Plan.Time` timestamp with height zero;
finalized plans returned to validator tooling have a positive height and no
time field. IBC upgrade plans remain height-only.

The existing `plan`, `applied`, and `module_versions` CLI commands retain their
outputs. Additive commands provide JSON without historical block RPC lookups:

```sh
nibiru query upgrade applied-record v2.22.0
nibiru query upgrade status
nibiru query upgrade params
nibiru query upgrade history --reverse --limit 20
```

`status` includes a terminal selection-failure receipt if one exists. A failed
request requires another approved scheduling request; there is no timed retry
scheduler. Unknown native faults and failed native migrations stop before
commit instead of being converted to scheduling failures.
