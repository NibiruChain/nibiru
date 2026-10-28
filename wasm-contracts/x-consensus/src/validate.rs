//! validate.rs: Validation of [`ConsensusParams`].
//!
//! The standard fields follow `ConsensusParams.ValidateBasic` in
//! `bft/types/params.go` (Nibiru BFT, CometBFT v0.37.18) rule for rule and in
//! the same order, so the contract never stores params that the engine would
//! reject, and the first error reported matches the one Go would report.
//! Line numbers below refer to that file at main `ef8454ae3`.

use crate::error::ContractError;
use crate::msg::{ConsensusParams, MAX_MIN_BLOCK_COMMIT_DURATION_MS};

/// Largest allowed `block.max_bytes`, 100 MiB. Mirrors `MaxBlockSizeBytes`
/// (`bft/types/params.go:16`).
pub const MAX_BLOCK_SIZE_BYTES: i64 = 104_857_600;

/// ABCI public key type names the engine knows. Mirrors the keys of
/// `ABCIPubKeyTypesToNames` (`bft/types/params.go:28-31`), whose values are
/// `ed25519.KeyType` (`bft/crypto/ed25519/ed25519.go:34`) and
/// `secp256k1.KeyType` (`bft/crypto/secp256k1/secp256k1.go:24`).
pub const KNOWN_PUB_KEY_TYPES: [&str; 2] = ["ed25519", "secp256k1"];

/// Shorthand for an [`ContractError::InvalidParams`] result.
fn invalid<T>(msg: String) -> Result<T, ContractError> {
    Err(ContractError::InvalidParams(msg))
}

impl ConsensusParams {
    /// Checks every rule and returns the first one broken.
    ///
    /// Pure function of the params, so the contract can run it on every write
    /// and tests can call it directly. Duplicate key types are allowed, as in
    /// Go.
    pub fn validate(&self) -> Result<(), ContractError> {
        let block_max_bytes = self.block.max_bytes.i64();
        let block_max_gas = self.block.max_gas.i64();
        let ev_max_age_num_blocks = self.evidence.max_age_num_blocks.i64();
        let ev_max_age_duration_ns = self.evidence.max_age_duration_ns.i64();
        let ev_max_bytes = self.evidence.max_bytes.i64();

        // bft/types/params.go:119-122
        if block_max_bytes <= 0 {
            return invalid(format!(
                "block.max_bytes must be greater than 0. Got {block_max_bytes}"
            ));
        }
        // bft/types/params.go:123-126
        if block_max_bytes > MAX_BLOCK_SIZE_BYTES {
            return invalid(format!(
                "block.max_bytes is too big. \
                 {block_max_bytes} > {MAX_BLOCK_SIZE_BYTES}"
            ));
        }
        // bft/types/params.go:128-131
        if block_max_gas < -1 {
            return invalid(format!(
                "block.max_gas must be greater or equal to -1. \
                 Got {block_max_gas}"
            ));
        }
        // bft/types/params.go:133-136
        if ev_max_age_num_blocks <= 0 {
            return invalid(format!(
                "evidence.max_age_num_blocks must be greater than 0. \
                 Got {ev_max_age_num_blocks}"
            ));
        }
        // bft/types/params.go:138-141
        if ev_max_age_duration_ns <= 0 {
            return invalid(format!(
                "evidence.max_age_duration_ns must be greater than 0. \
                 Got {ev_max_age_duration_ns}"
            ));
        }
        // bft/types/params.go:143-146
        if ev_max_bytes > block_max_bytes {
            return invalid(format!(
                "evidence.max_bytes is greater than upper bound, \
                 {ev_max_bytes} > {block_max_bytes}"
            ));
        }
        // bft/types/params.go:148-151
        if ev_max_bytes < 0 {
            return invalid(format!(
                "evidence.max_bytes must be non negative. Got: {ev_max_bytes}"
            ));
        }
        // bft/types/params.go:153-155
        if self.validator.pub_key_types.is_empty() {
            return invalid(
                "len(validator.pub_key_types) must be greater than 0"
                    .to_string(),
            );
        }
        // bft/types/params.go:157-164
        for (i, key_type) in self.validator.pub_key_types.iter().enumerate() {
            if !KNOWN_PUB_KEY_TYPES.contains(&key_type.as_str()) {
                return invalid(format!(
                    "validator.pub_key_types[{i}], {key_type}, \
                     is an unknown pubkey type"
                ));
            }
        }
        // Not in Go: the chain-wide commit timing added by x-consensus. The
        // lower bound 0 is implied by u64.
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
