use cosmwasm_schema::cw_serde;

use crate::msg::VestingSchedule;
use cosmwasm_std::Uint128;
use cw20::Denom;
use cw_storage_plus::Map;

pub const VESTING_ACCOUNTS: Map<(&str, &str), VestingAccount> =
    Map::new("vesting_accounts");

#[cw_serde]
pub struct VestingAccount {
    pub master_address: Option<String>,
    pub address: String,
    pub vesting_denom: Denom,
    pub vesting_amount: Uint128,
    pub vesting_schedule: VestingSchedule,
    pub claimed_amount: Uint128,
}

pub fn denom_to_key(denom: Denom) -> String {
    match denom {
        Denom::Cw20(addr) => format!("cw20-{}", addr),
        Denom::Native(denom) => format!("native-{}", denom),
    }
}

#[test]
fn test_denom_to_key() {
    use cosmwasm_std::Uint64;

    let schedule = VestingSchedule::LinearVesting {
        start_time: Uint64::new(100),
        end_time: Uint64::new(120),
        vesting_amount: Uint128::new(1000),
    };

    let vesting_account = VestingAccount {
        master_address: None,
        address: String::from("address"),
        vesting_denom: Denom::Native(String::from("nibi")),
        vesting_amount: Uint128::zero(),
        vesting_schedule: schedule,
        claimed_amount: Uint128::zero(),
    };

    assert_eq!(denom_to_key(vesting_account.vesting_denom), "native-nibi");
}
