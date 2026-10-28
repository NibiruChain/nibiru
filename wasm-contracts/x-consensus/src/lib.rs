//! x-consensus: the on-chain owner of Nibiru BFT's consensus params.
//!
//! Nibiru BFT, Nibiru's fork of CometBFT, gets its consensus params (block
//! size and gas limits, evidence limits, allowed validator key types) from
//! the app. Before x-consensus, the x/consensus module owned them: a gov
//! proposal sent its `MsgUpdateParams` and the module wrote its store. This
//! contract takes over that role. It holds the block, evidence and validator
//! groups of `tendermint.types.ConsensusParams`, plus
//! `min_block_commit_duration_ms`, a chain-wide commit wait that replaces
//! each node's `config.toml` `timeout_commit`.
//!
//! The x/consensus store still exists, but only as the per-block copy that
//! baseapp and Nibiru BFT read. Its `MsgUpdateParams` always fails
//! (`x/consensus/module.go`).
//!
//! # Who can change it
//!
//! Only the current x/sudo root. The contract asks x/sudo for the root on
//! every execute ([`sudo::query_sudo_root`]) and stores no owner, so a root
//! rotation through x/sudo `MsgChangeRoot` takes effect at once.
//!
//! # When a change takes effect
//!
//! The x/consensus EndBlocker runs after every other module's. It queries
//! this contract under a fixed gas limit, validates the result with Nibiru
//! BFT's own rules, and copies it into the x/consensus store. baseapp returns
//! that copy to Nibiru BFT from the same EndBlock. So a change executed in
//! block H applies from block H+1, for the app and the engine together. If
//! the query fails or the result is invalid, the store keeps its last good
//! params and the chain keeps running.
//!
//! # Migration
//!
//! The contract is its own Wasm admin. The x/sudo root migrates it with
//! [`msg::ExecuteMsg::Migrate`], which the contract turns into a Wasm migrate
//! of itself, so migration authority follows the live root just as param
//! updates do. Governance stays the break-glass: Nibiru's Wasm keeper lets a
//! gov proposal migrate any contract regardless of its admin
//! (`x/wasm/keeper/authz_policy.go:66`).

pub mod contract;
pub mod error;
pub mod msg;
pub mod state;
pub mod sudo;
pub mod validate;

#[cfg(test)]
mod tests;
