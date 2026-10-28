//! x-consensus: on-chain store for Nibiru BFT consensus params.
//!
//! This contract replaces the storage role of the x/consensus module. It
//! holds every field of `tendermint.types.ConsensusParams` plus the
//! chain-wide `min_block_commit_duration_ms`. A Go adapter in `nibid` queries
//! it once per block at EndBlock and feeds the result to baseapp and to the
//! engine, so a change applies from the next block. Keeping the values and
//! their rules in a contract lets the owner (the x/sudo root) change them by
//! contract execution, and the rules by contract migration, without a chain
//! upgrade.

pub mod contract;
pub mod error;
pub mod msg;
pub mod state;
pub mod validate;

#[cfg(test)]
mod tests;
