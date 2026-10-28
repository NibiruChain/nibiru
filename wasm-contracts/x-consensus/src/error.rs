//! error.rs: Errors returned by the x-consensus contract.

use cosmwasm_std::StdError;
use thiserror::Error;

/// Every error the contract returns. Any error fails the whole message, so
/// the stored params and the Wasm admin stay as they were.
#[derive(Error, Debug, PartialEq)]
pub enum ContractError {
    /// Storage, serialization, address or querier failure. This includes a
    /// failed x/sudo root query, which rejects the message rather than
    /// skipping the root check, and an invalid `UpdateAdmin` address.
    #[error("{0}")]
    Std(#[from] StdError),

    /// The sender is not the x/sudo root. Only the root may change state.
    /// The error names both addresses so a failed proposal is easy to
    /// diagnose, for example after a root rotation.
    #[error("unauthorized: sender {sender} is not the x/sudo root {root}")]
    Unauthorized {
        /// Address that sent the message.
        sender: String,
        /// Current x/sudo root.
        root: String,
    },

    /// x/sudo has no root, so nobody may change state. This is the x/sudo
    /// genesis default until a root is set. Kept apart from
    /// [`ContractError::Unauthorized`] so an empty root is never compared
    /// against a sender.
    #[error("unauthorized: x/sudo has no root set")]
    NoSudoRoot,

    /// The params break a validation rule. The message names the field and
    /// follows the Go error text in `bft/types/params.go` where one exists.
    #[error("invalid consensus params: {0}")]
    InvalidParams(String),
}
