// Package "evm/evmstate" implements a go-ethereum [vm.StateDB] for Nibiru's
// Multi VM stack, implemented as [SDB]. The [SDB] manages EVM *and* non-EVM
// state reachable via the [sdk.Context] by branching the world state with
// "CacheMultiStore" snapshots. There are no journal entries.
//
// ### Model:
//   - Each [SDB] represents one Ethereum transaction's execution scope.
//   - Snapshot() creates a new world-state branch (cached multistore) and a
//     fresh LocalState layer for EVM-specific metadata (logs, refunds, access
//     list, transient storage).
//   - RevertToSnapshot(n) restores the exact prior world state by jumping to snapshot n.
//   - Commit() writes cached branches back toward the root Context; BaseApp
//     later commits the root.
//
// ### Notes:
//   - Nibiru uses IAVL-backed KV stores, not MPT/Verkle; we don't compute
//     Ethereum storage roots.
//   - Read paths (balances/accounts) consult the active branched Context; no
//     journals are needed.
//   - This design mirrors database snapshotting: revert == restore snapshot, not
//     "undo logs".
package evmstate
