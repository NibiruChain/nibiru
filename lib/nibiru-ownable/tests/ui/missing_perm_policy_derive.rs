use cosmwasm_schema::cw_serde;
use nibiru_ownable::ownable_execute;

#[ownable_execute(perms)]
#[cw_serde]
enum ExecuteMsg {
    Public {},
}

fn main() {}
