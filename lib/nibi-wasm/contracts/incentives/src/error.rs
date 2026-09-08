use cosmwasm_std::StdError;
use thiserror::Error;

#[allow(dead_code)]
#[derive(Error, Debug, PartialEq)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("not implemented")]
    NotImplemented,

    #[error("unknown request")]
    UnknownRequest,

    #[error("no rewards to withdraw: {0}")]
    NothingToWithdraw(String),

    #[error("funds required")]
    FundsRequired,

    #[error("epoch out of bounds for program {1}: {0}")]
    EpochOutOfBounds(u64, u64),

    #[error("epoch {0} for program {1} can be processed after block {2}")]
    EpochProcessBlock(u64, u64, u64),

    #[error(
        "incentives program has finished at block {1} (current block: {2}): {0}"
    )]
    ProgramFinished(u64, u64, u64),
}
