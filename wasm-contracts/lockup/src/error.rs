use cosmwasm_std::StdError;
use thiserror::Error;

#[derive(Error, Debug, PartialEq)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("not implemented")]
    NotImplemented,

    #[error("unknown request")]
    UnknownRequest,

    #[error("invalid lock duration")]
    InvalidLockDuration,

    #[error("invalid coins: {0}")]
    InvalidCoins(String),

    #[error("lock not found: {0}")]
    NotFound(u64),

    #[error("already unlocking: {0}")]
    AlreadyUnlocking(u64),

    #[error("funds already withdrawn: {0}")]
    FundsAlreadyWithdrawn(u64),

    #[error("not matured: {0}")]
    NotMatured(u64),
}
