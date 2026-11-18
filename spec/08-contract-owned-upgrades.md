# Contract-Owned Upgrades

Status: Implementation specification for the unreleased v2.22.0 upgrade.

Nibiru's `x/upgrade` preserves the native Cosmos upgrade interface while using
the Rust `x-upgrade` contract for scheduling rules and internal state. The
enabling release is **v2.22.0**. This is an implementation change behind the
existing governance workflow, with an additive time-target scheduling feature.

## Ownership and Permissions

The contract owns the pending request, completed-upgrade history, module-version
map, and protocol version. Go retains registered migration functions, startup
store loaders, the local upgrade-info file, BaseApp protocol integration, and
IBC proof-visible records. There is no second native copy of the four ordinary
state families.

| Operation | Entry Point and Authority |
| --- | --- |
| Schedule, replace, cancel | Native governance message adapter, then contract `execute` as the authenticated governance authority |
| Runtime configuration | Contract `execute`, checking the current `x/sudo` root |
| Contract code migration or admin transfer | Root-authorized `execute`; self-admin Wasm migration follows the consensus contract pattern |
| Height selection and native completion/version bookkeeping | Contract `sudo` from trusted native host code |

The native handler verifies authority before passing a caller address to Wasm.
Rust then checks that authority for scheduling operations. A caller-supplied
address alone is not authentication. Both modern messages and legacy proposal
handlers use the contract-backed keeper.

Contract `sudo` has no sender argument. It trusts native host dispatch, including
the root-controlled hook infrastructure. This is intentional: the same root can
already replace the contract's code. A reserved module-caller address would add
plumbing without isolating the contract from authorities already trusted to
replace its rules. Governance retains its exceptional Wasm migration path.

## Scheduling and Compatibility

One pending request is supported. A valid new governance-approved request
replaces it, retaining the existing module behavior. Cancellation remains
governance-authorized; no queue or generic scheduling framework is introduced.

Height requests keep their existing meaning. Time requests reuse protobuf field
`Plan.Time` as a request-only absolute timestamp and require `Height == 0`.
Exactly one target form is accepted. A separate scheduling-request validator
supports this addition while ordinary `Plan.ValidateBasic` stays height-only;
IBC upgrade proposals therefore remain height-only too.

```text
Governance executes a scheduling request
  -> validate against execution context and store the request in Rust
  -> an already overdue timestamp returns a normal message error

First BeginBlock at height H with block time >= target
  -> sample activation_delay_blocks from contract configuration
  -> persist a concrete execution height H + delay
  -> expose the ordinary native CurrentPlan response

Concrete execution height
  -> existing missing-handler halt and upgrade-info.json behavior
  -> existing binary replacement and native migration handler
```

The default delay is 1,000 blocks, with a minimum of two. It is consensus state,
not operator configuration. After selection the height is fixed; later delay
changes do not move it. The delay provides a block cushion, not a guaranteed
duration in seconds. Compare consensus timestamps, never local clocks.

Native finalized plans and the local upgrade-info file leave deprecated time
unset. An unresolved time request must not masquerade as height zero or an
estimated executable plan. Existing queries and CLI behavior remain supported;
additional CLI views expose status, configuration, applied records, and history.

Module-version writes retain update-only semantics for omitted names. History
uses height/name ordering plus a nonunique name index, preserving legacy
duplicate-name lookup and tie behavior. Optional state remains optional; absence
is not silently materialized as a new stored default.

## Enabling Cutover and Restoration

The v2.22.0 handler runs normal migrations with legacy accessors active. It then
imports the current legacy snapshot into the contract, verifies the result,
deletes only the four replaced key families, and writes the activation marker
last in a cached context. It does not preempt outer upgrade bookkeeping.

```text
Legacy plan/history/version reads
  -> native handler runs module migrations
  -> snapshot and verify contract import
  -> activate contract backend
  -> return updated module versions
ApplyUpgrade resumes through fresh keeper calls
  -> write returned module versions to Rust
  -> increment protocol version once and update BaseApp
  -> clear executed plan and normal IBC upgrade state
  -> record completion once
```

No raw store handle spans the handler in the existing path. Consequently the
backend switch can redirect trailing writes without an EndBlock finalizer.
An EndBlock cutover is a fallback only if testing demonstrates a concrete need.
Historical contexts without an activation marker continue reading legacy state.
IBC-only test applications can remain in legacy mode without adding Wasm.

Fresh genesis initializes Wasm before creating the upgrade contract. Imported
genesis preserves its contract records instead of replacing them with binary
defaults. The host reconstructs the activation address and synchronizes the
required native protocol state. Wasm exports contract code, metadata, and raw
key/value records; Go handlers and local files are separate host integration.

## Terminal Selection Failure

Normal validation errors during proposal execution use existing governance
rollback and failed-proposal behavior. A deterministic failure later, while
selecting an execution height, is a failed upgrade request: the earlier
governance proposal remains passed.

Selection runs under a fixed 1,000,000 SDK gas limit in a child cache. Explicit
contract-result errors and bounded gas exhaustion discard partial writes and
events. The host records one terminal failure receipt outside that failed cache,
identifying the request, height, and a stable error category. It emits one
failure event and stops selecting that request. Replacement or cancellation
clears suppression atomically. There are no retry timers or counters.

The receipt is the exceptional native record needed when the contract cannot
write its own failure status. It is exported/imported by upgrade genesis, and
the CLI combines it with contract status by request ID. Arbitrary runtime error
text stays in logs, not consensus state.

Unknown VM/host/storage errors and failures during native migration remain
fatal before block commit. An unreadable pending identity is not fabricated
into a failed request. Selection reads only its own contract state; it does not
turn external query failures into consensus failure receipts.

## Acceptance Evidence

The acceptance harness lives in `contrib/upgrade-e2e`. It uses disposable homes,
explicit ports, pinned artifacts, bounded waits, and run-owned process cleanup.
It must not reset an operator's existing node home.

Required evidence includes a released old binary upgraded through real
governance to the candidate under Cosmovisor, followed by a time-targeted
successor upgrade. The successor handler is test-only because the normal
early-binary guard prohibits registering a future handler in the waiting binary.

Check state conversion, trailing writes, permissions, terminal failure,
cancellation/replacement, restart, replay, and nonzero-height genesis round trips.
Use historical proposal and upgrade events as compatibility fixtures. Exercise
Cosmopilot discovery and stale-plan behavior separately; do not change chain
cancellation semantics to conceal a controller defect.

## Contract Schema and Release Review

Contract JSON identifies consensus heights as `block_height`, scheduling
permissions as `gov_authority`, and absolute time targets as
`target_time_unix_nanos`. Nanoseconds preserve consensus block-time precision;
the native protobuf API retains `height` and request-only `Plan.Time`.
See the x-upgrade README for operator query examples and indexed-history
semantics. Native code translates the schema at the existing API boundary.

The typed live-root query uses regenerated `nibiru-std` sudo definitions from
the in-tree protos, including role memberships. Regenerate with command
`just rs proto-gen --path proto/nibiru/sudo/v1`, which also runs the Rust import
cleanup. Do not substitute stale generated ownership types or a cached root.

A failed x-upgrade import is a required migration failure. Returning an error
prevents completion of a partially enabled release. A log-and-continue policy
would deliberately deploy a height-only fallback without the promised contract
state and time scheduling; that policy is not part of this release.
