use cosmwasm_schema::cw_serde;

#[cw_serde]
pub struct InstantiateMsg {}

#[cw_serde]
pub enum ExecuteMsg {
    Lock { blocks: u64 },

    InitiateUnlock { id: u64 },

    WithdrawFunds { id: u64 },
}

#[cw_serde]
pub enum QueryMsg {
    LocksByDenomUnlockingAfter {
        denom: String,
        unlocking_after: u64,
    },
    LocksByDenomAndAddressUnlockingAfter {
        denom: String,
        unlocking_after: u64,
        address: String,
    },
    LocksByDenomBetween {
        denom: String,
        locked_before: u64,
        unlocking_after: u64,
    },
    LocksByDenomAndAddressBetween {
        denom: String,
        address: String,
        locked_before: u64,
        unlocking_after: u64,
    },
}
