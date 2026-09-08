use std::collections::HashSet;

use cosmwasm_schema::cw_serde;

use crate::msg::VestingSchedule;
use cosmwasm_std::{StdResult, Timestamp, Uint128};
use cw_storage_plus::{Item, Map};

pub const VESTING_ACCOUNTS: Map<&str, VestingAccount> =
    Map::new("vesting_accounts");
pub const UNALLOCATED_AMOUNT: Item<Uint128> = Item::new("unallocated_amount");
pub const DENOM: Item<String> = Item::new("denom");
pub const WHITELIST: Item<Whitelist> = Item::new("whitelist");

#[cw_serde]
pub struct Whitelist {
    pub members: HashSet<String>,
    pub admin: String,
}

impl Whitelist {
    pub fn is_admin(&self, addr: impl AsRef<str>) -> bool {
        let addr = addr.as_ref();
        self.admin == addr
    }

    pub fn is_member(&self, addr: impl AsRef<str>) -> bool {
        let addr = addr.as_ref();
        self.members.contains(addr)
    }
}

#[cw_serde]
pub struct VestingAccount {
    pub address: String,
    pub vesting_amount: Uint128,
    pub cliff_amount: Uint128,
    pub vesting_schedule: VestingSchedule,
    pub claimed_amount: Uint128,
}

impl VestingAccount {
    pub fn vested_amount(&self, block_time: Timestamp) -> StdResult<Uint128> {
        match self.vesting_schedule {
            VestingSchedule::LinearVestingWithCliff {
                start_time: _start_time,
                end_time,
                cliff_time,
            } => {
                if block_time.seconds() < cliff_time.u64() {
                    return Ok(Uint128::zero());
                }

                if block_time.seconds() == cliff_time.u64() {
                    return Ok(self.cliff_amount);
                }

                if block_time.seconds() >= end_time.u64() {
                    return Ok(self.vesting_amount);
                }

                let remaining_token =
                    self.vesting_amount.checked_sub(self.cliff_amount)?;
                let vested_token = remaining_token
                    .checked_mul(Uint128::from(
                        block_time.seconds() - cliff_time.u64(),
                    ))?
                    .checked_div(Uint128::from(end_time - cliff_time))?;

                Ok(vested_token + self.cliff_amount)
            }
        }
    }
}
