use cosmwasm_schema::cw_serde;
use cosmwasm_std::Addr;
use cw_storage_plus::Item;

pub const STATE: Item<State> = Item::new("state");

#[cw_serde]
pub struct State {
    pub count: i64,
    pub owner: Addr,
}
