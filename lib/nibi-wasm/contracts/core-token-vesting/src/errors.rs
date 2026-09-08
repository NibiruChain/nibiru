use thiserror::Error;

#[derive(Error, Debug, PartialEq)]
pub enum ContractError {
    #[error(transparent)]
    Std(#[from] cosmwasm_std::StdError),

    #[error(transparent)]
    Vesting(#[from] VestingError),

    #[error(transparent)]
    Cliff(#[from] CliffError),

    #[error(transparent)]
    Overflow(#[from] cosmwasm_std::OverflowError),
}

#[derive(thiserror::Error, Debug, PartialEq)]
pub enum CliffError {
    #[error("cliff_amount is zero but should be greater than 0")]
    ZeroAmount,

    #[error("cliff_time ({cliff_time}) should be greater than block_time ({block_time})")]
    InvalidTime { cliff_time: u64, block_time: u64 },

    #[error("cliff_amount ({cliff_amount}) should be less than or equal to vesting_amount ({vesting_amount})")]
    ExcessiveAmount {
        cliff_amount: u128,
        vesting_amount: u128,
    },
}

#[derive(thiserror::Error, Debug, PartialEq)]
pub enum VestingError {
    #[error("vesting_amount is zero but should be greater than 0")]
    ZeroVestingAmount,

    #[error(
        "end_time ({end_time}) should be greater than start_time ({start_time})"
    )]
    InvalidTimeRange { start_time: u64, end_time: u64 },

    #[error(transparent)]
    Cliff(#[from] CliffError),

    #[error("vesting_amount ({vesting_amount}) should be equal to deposit_amount ({deposit_amount})")]
    MismatchedVestingAndDepositAmount {
        vesting_amount: u128,
        deposit_amount: u128,
    },
}
