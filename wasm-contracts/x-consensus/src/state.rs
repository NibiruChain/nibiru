//! state.rs: Storage layout of the x-consensus contract.

use cw_storage_plus::Item;

use crate::msg::ConsensusParams;

/// The stored consensus params. Always holds a value that passed
/// [`ConsensusParams::validate`], because every write goes through it.
/// There is no owner item: the x/sudo root is read at execute time.
pub const PARAMS: Item<ConsensusParams> = Item::new("params");
