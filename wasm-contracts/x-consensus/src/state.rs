//! state.rs: Storage layout of the x-consensus contract.

use cw_storage_plus::Item;

use crate::msg::ConsensusParams;

/// The stored consensus params, under the raw key `params`.
///
/// Every write goes through [`ConsensusParams::validate`], so the stored
/// value always passed validation. There is no owner item, because the
/// x/sudo root is read at execute time, and no cw2 version item, because the
/// code checksum identifies the version. This is the contract's only state.
pub const PARAMS: Item<ConsensusParams> = Item::new("params");
