//! error.rs: Errors returned by the x-consensus contract.

use cosmwasm_std::StdError;
use thiserror::Error;

/// Every error the contract returns.
#[derive(Error, Debug, PartialEq)]
pub enum ContractError {
    /// Storage, serialization, or querier failure, including a failed
    /// x/sudo root query.
    #[error("{0}")]
    Std(#[from] StdError),

    /// The sender is not the x/sudo root. Only the root may change state.
    #[error("unauthorized: sender {sender} is not the x/sudo root {root}")]
    Unauthorized {
        /// Address that sent the message.
        sender: String,
        /// Current x/sudo root.
        root: String,
    },

    /// x/sudo has no root, so nobody may change state. This is the x/sudo
    /// genesis default until a root is set.
    #[error("unauthorized: x/sudo has no root set")]
    NoSudoRoot,

    /// The params break a validation rule. The message names the field and
    /// mirrors the Go error text in `bft/types/params.go` where one exists.
    #[error("invalid consensus params: {0}")]
    InvalidParams(String),
}
