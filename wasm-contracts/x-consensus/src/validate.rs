//! validate.rs: Validation of [`ConsensusParams`].
//!
//! The standard fields follow `ConsensusParams.ValidateBasic` in
//! `bft/types/params.go` (Nibiru BFT, forked from CometBFT v0.37.18) rule for
//! rule and in the same order. Nibiru BFT runs that check on every params
//! update it receives and stops the chain if it fails
//! (`bft/state/execution.go:524`), so the contract must never store params
//! that would fail it. Matching the order also means the first error
//! reported here is the one Go would report. Line numbers below refer to
//! `bft/types/params.go` in this tree.
//!
//! The x/consensus EndBlocker runs the Go check again before copying the
//! params, so a bug here or a bad migration cannot halt the chain either.

use crate::error::ContractError;
use crate::msg::{ConsensusParams, MAX_MIN_BLOCK_COMMIT_DURATION_MS};

/// Largest allowed `block.max_bytes`, 100 MiB. Mirrors `MaxBlockSizeBytes`
/// (`bft/types/params.go:16`).
pub const MAX_BLOCK_SIZE_BYTES: i64 = 104_857_600;

/// ABCI public key type names Nibiru BFT knows. Mirrors the keys of
/// `ABCIPubKeyTypesToNames` (`bft/types/params.go:28-31`), whose values are
/// `ed25519.KeyType` (`bft/crypto/ed25519/ed25519.go:34`) and
/// `secp256k1.KeyType` (`bft/crypto/secp256k1/secp256k1.go:24`).
pub const KNOWN_PUB_KEY_TYPES: [&str; 2] = ["ed25519", "secp256k1"];

/// Shorthand for a [`ContractError::InvalidParams`] result.
fn invalid<T>(msg: String) -> Result<T, ContractError> {
    Err(ContractError::InvalidParams(msg))
}

impl ConsensusParams {
    /// Checks every rule and returns the first one broken.
    ///
    /// A pure function of the params, so instantiate and every update run
    /// the same check and tests can call it directly. It checks only what
    /// Nibiru BFT checks, plus the commit wait bound: duplicate key types
    /// are allowed, as in Go, and a `block.max_bytes` too small to hold a
    /// block header passes, as in Go.
    pub fn validate(&self) -> Result<(), ContractError> {
        let block_max_bytes = self.block.max_bytes.i64();
        let block_max_gas = self.block.max_gas.i64();
        let ev_max_age_num_blocks = self.evidence.max_age_num_blocks.i64();
        let ev_max_age_duration_ns = self.evidence.max_age_duration_ns.i64();
        let ev_max_bytes = self.evidence.max_bytes.i64();

        // bft/types/params.go:118-121
        if block_max_bytes <= 0 {
            return invalid(format!(
                "block.max_bytes must be greater than 0. Got {block_max_bytes}"
            ));
        }
        // bft/types/params.go:122-125
        if block_max_bytes > MAX_BLOCK_SIZE_BYTES {
            return invalid(format!(
                "block.max_bytes is too big. \
                 {block_max_bytes} > {MAX_BLOCK_SIZE_BYTES}"
            ));
        }
        // bft/types/params.go:127-130
        if block_max_gas < -1 {
            return invalid(format!(
                "block.max_gas must be greater or equal to -1. \
                 Got {block_max_gas}"
            ));
        }
        // bft/types/params.go:132-135
        if ev_max_age_num_blocks <= 0 {
            return invalid(format!(
                "evidence.max_age_num_blocks must be greater than 0. \
                 Got {ev_max_age_num_blocks}"
            ));
        }
        // bft/types/params.go:137-140
        if ev_max_age_duration_ns <= 0 {
            return invalid(format!(
                "evidence.max_age_duration_ns must be greater than 0. \
                 Got {ev_max_age_duration_ns}"
            ));
        }
        // bft/types/params.go:142-145
        if ev_max_bytes > block_max_bytes {
            return invalid(format!(
                "evidence.max_bytes is greater than upper bound, \
                 {ev_max_bytes} > {block_max_bytes}"
            ));
        }
        // bft/types/params.go:147-150
        if ev_max_bytes < 0 {
            return invalid(format!(
                "evidence.max_bytes must be non negative. Got: {ev_max_bytes}"
            ));
        }
        // bft/types/params.go:152-154
        if self.validator.pub_key_types.is_empty() {
            return invalid(
                "len(validator.pub_key_types) must be greater than 0"
                    .to_string(),
            );
        }
        // bft/types/params.go:157-163
        for (i, key_type) in self.validator.pub_key_types.iter().enumerate() {
            if !KNOWN_PUB_KEY_TYPES.contains(&key_type.as_str()) {
                return invalid(format!(
                    "validator.pub_key_types[{i}], {key_type}, \
                     is an unknown pubkey type"
                ));
            }
        }
        // Not in Go: the chain-wide commit wait added by x-consensus. The
        // lower bound 0 is implied by u64. The Go module enforces the same
        // cap (MaxMinBlockCommitDurationMs).
        if let Some(ms) = self.min_block_commit_duration_ms {
            if ms > MAX_MIN_BLOCK_COMMIT_DURATION_MS {
                return invalid(format!(
                    "min_block_commit_duration_ms must be at most \
                     {MAX_MIN_BLOCK_COMMIT_DURATION_MS}. Got {ms}"
                ));
            }
        }
        Ok(())
    }
}
