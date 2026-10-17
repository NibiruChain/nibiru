use cosmwasm_schema::cw_serde;
use cosmwasm_std::Addr;

#[cw_serde]
pub struct InstantiateMsg {
    pub lockup_contract_address: Addr,
}

#[cw_serde]
pub enum QueryMsg {
    ProgramFunding { program_id: u64 },
    EpochInfo { program_id: u64, epoch_number: u64 },
}

#[cw_serde]
pub enum ExecuteMsg {
    CreateProgram {
        denom: String,
        epochs: u64,
        epoch_block_duration: u64,
        min_lockup_blocks: u64,
    },

    FundProgram {
        id: u64,
    },

    ProcessEpoch {
        id: u64,
    },

    WithdrawRewards {
        id: u64,
    },
}
