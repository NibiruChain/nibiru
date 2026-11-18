# Upgrade Acceptance Runner

This disposable localnet exercises the upgrade module through governance and
Cosmovisor. It preserves the run directory on success and failure. It does not
use `~/.nibid`, replace globally installed binaries, or stop unrelated nodes.

The baseline is the published GitHub release `v2.21.2`. The runner verifies its
platform archive against the cached release checksum manifest before copying
the native executable. It uses Cosmovisor `v1.7.1`, with downloads disabled and
the upgrade binaries staged ahead of time. Backups are disabled only inside
this disposable experiment to avoid copying its database twice per upgrade.

## Commands

Install `uv`, then run these commands from the repository root. The recipes
use the local Python project and lockfile; tests run with pytest.

```sh
just --justfile contrib/upgrade-e2e/justfile test
just --justfile contrib/upgrade-e2e/justfile prepare
just --justfile contrib/upgrade-e2e/justfile smoke
```

Mode `smoke` also submits and passes a native governance scheduling proposal,
then passes its cancellation, using only the published baseline binary.

The checksum-verified release cache is discovered in this worktree or its main
checkout. Option `--baseline-cache DIRECTORY` selects a different copy of the
same cached release, containing its archives and checksum manifest.

Build two candidate binaries from the same commit:

```sh
VERSION=v2.22.0 BUILDDIR="$PWD/.cache/upgrade-e2e/build-enabling" just build
VERSION=v2.22.0-e2e-next \
  BUILDDIR="$PWD/.cache/upgrade-e2e/build-successor" \
  NIBID_EXTRA_TAGS=nibiru_upgrade_e2e_next just build

just --justfile contrib/upgrade-e2e/justfile run \
  --candidate "$PWD/.cache/upgrade-e2e/build-enabling/nibid" \
  --successor "$PWD/.cache/upgrade-e2e/build-successor/nibid"
```

The successor build tag registers the fixture handler `e2e-x-upgrade-next`.
It is excluded from normal builds. The enabling binary must not contain that
handler: The upgrade module correctly rejects installing a future upgrade's
handler before its execution height.

## What the Run Exercises

1. Initialize the old binary's state with a ten-second voting period and one
   validator holding all stake. The validator is also the local sudo root.
2. Pass a native height-based software-upgrade proposal for `v2.22.0`; observe
   the old binary halt and Cosmovisor install the enabling candidate.
3. Query contract-backed status, history, and module versions. Schedule and
   cancel a plan, then replace another plan with a time-targeted request.
4. Verify that the time request selects a concrete height exactly 1,000 blocks
   after the first block reaching the target. Do not shorten the default delay
   merely to speed up this acceptance test.
5. Observe the second halt and switch to the successor candidate. Restart it,
   checking history and module-version stability.
6. Export state at its nonzero height, import into a fresh node home, and
   compare the history, module versions, and raw contract records after blocks
   resume. The old node is stopped before reusing its validator signing key.

The run takes roughly ten minutes at a half-second block interval, plus build
time. Option `--timeout` bounds height waits and defaults to thirty minutes.
The RPC, P2P, and gRPC ports default to 27657, 27656, and 27658; option
`--port-base` moves the entire set. Busy ports and existing run directories are
rejected. Cleanup signals only the process group started by the runner and
never removes the node homes or evidence.

## Evidence and Limits

The run prints its directory under `.cache/upgrade-e2e/`. Directory `evidence/`
contains binary versions and hashes, proposal documents, transaction receipts,
query snapshots, surrounding blocks/results, the upgrade-info files, exported
genesis, and a final JSON result. File `node.log` contains Cosmovisor and daemon
logs. Node homes contain disposable private keys; do not publish those homes.

A passing run proves the local governance/Cosmovisor path, restart, and genesis
round trip. It does not claim a live validator rollout, a Cosmopilot controller
test, or deterministic replay after an injected migration failure. Those need
their respective integration tests. Historical upgrade fixtures should be
compared with the saved proposal/block evidence when reviewing API and event
compatibility.

The runner assumes the candidate implements the contract-backed upgrade CLI
and the additive time request format. Run `smoke` independently while that
implementation is being built; its result is only a baseline environment check.
