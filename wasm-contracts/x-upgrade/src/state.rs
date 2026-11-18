use crate::msg::{CompletedUpgrade, Config, PendingSchedule};
use cosmwasm_std::Uint64;
use cw_storage_plus::{Index, IndexList, IndexedMap, Item, Map, MultiIndex};

pub const CONFIG: Item<Config> = Item::new("config");
pub const PENDING: Item<PendingSchedule> = Item::new("pending");
pub const NEXT_ID: Item<u64> = Item::new("next_id");
pub const PROTOCOL_VERSION: Item<Uint64> = Item::new("protocol_version");
pub const MODULE_VERSIONS: Map<&str, Uint64> = Map::new("module_versions");

/// Nonunique name lookup alongside chronological completion ordering. Imported
/// legacy history can contain repeated names, so this cannot be a unique index.
pub struct HistoryIndexes<'a> {
    /// Resolves the earliest completion for legacy applied-plan queries.
    pub name: MultiIndex<'a, String, CompletedUpgrade, (u64, String)>,
}
impl<'a> IndexList<CompletedUpgrade> for HistoryIndexes<'a> {
    fn get_indexes(
        &'_ self,
    ) -> Box<dyn Iterator<Item = &'_ dyn Index<CompletedUpgrade>> + '_> {
        Box::new(vec![&self.name as &dyn Index<CompletedUpgrade>].into_iter())
    }
}
/// Stores completions by (block_height, name) for stable forward/reverse pages.
pub fn history<'a>(
) -> IndexedMap<'a, (u64, String), CompletedUpgrade, HistoryIndexes<'a>> {
    IndexedMap::new(
        "completed",
        HistoryIndexes {
            name: MultiIndex::new(
                |_pk, record| record.name.clone(),
                "completed",
                "completed_by_name",
            ),
        },
    )
}
