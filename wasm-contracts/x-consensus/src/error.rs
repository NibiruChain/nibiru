//! error.rs: Errors returned by the x-consensus contract.

use cosmwasm_std::StdError;
use thiserror::Error;

/// Every error the contract returns.
#[derive(Error, Debug, PartialEq)]
pub enum ContractError {
    /// Storage, serialization, or address validation failure.
    #[error("{0}")]
    Std(#[from] StdError),

    /// The sender is not the owner. Only the owner may change state.
    #[error("unauthorized: sender {sender} is not the owner {owner}")]
    Unauthorized {
        /// Address that sent the message.
        sender: String,
        /// Current owner.
        owner: String,
    },

    /// The params break a validation rule. The message names the field and
    /// mirrors the Go error text in `bft/types/params.go` where one exists.
    #[error("invalid consensus params: {0}")]
    InvalidParams(String),
}
