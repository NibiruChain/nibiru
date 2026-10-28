//! state.rs: Storage layout of the x-consensus contract.

use cosmwasm_std::Addr;
use cw_storage_plus::Item;

use crate::msg::ConsensusParams;

/// The stored consensus params. Always holds a value that passed
/// [`ConsensusParams::validate`], because every write goes through it.
pub const PARAMS: Item<ConsensusParams> = Item::new("params");

/// The only address allowed to execute state changes.
pub const OWNER: Item<Addr> = Item::new("owner");
