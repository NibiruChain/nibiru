use cosmwasm_schema::cw_serde;
use cw_storage_plus::Item;

use crate::msg::WasmSudoMsg;

pub const STATE: Item<State> = Item::new("state");

#[cw_serde]
pub struct State {
    pub count: u64,
    pub query_error: bool,
    pub wasm_sudo_msg_calls: Vec<WasmSudoMsg>,
    pub last_sudo: Option<String>,
}
