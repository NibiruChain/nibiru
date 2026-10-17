use cosmwasm_std::{Addr, Coin, Uint128};
use cw_storage_plus::{Index, IndexList, IndexedMap, Item, Map, MultiIndex};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, JsonSchema, Clone, Debug)]
pub struct Funding {
    pub id: u64,
    pub program_id: u64,
    pub pay_from_epoch: u64,
    pub denom: String,
    pub initial_amount: Uint128,
    pub to_pay_each_epoch: Uint128, // how much needs to be paid each epoch
}

#[derive(Serialize, Deserialize, JsonSchema, Clone, Debug)]
pub struct Program {
    pub id: u64,
    pub epochs: u64,         // how many epochs
    pub epoch_duration: u64, // how many blocks in each epoch
    pub min_lockup_duration_blocks: u64,
    pub lockup_denom: String,
    pub start_block: u64,
    pub end_block: u64,
}
#[derive(Serialize, Deserialize, JsonSchema, Clone, Debug)]
pub struct EpochInfo {
    pub epoch_identifier: u64,
    pub for_coins_locked_before: u64,
    pub for_coins_unlocking_after: u64,
    pub to_distribute: Vec<Coin>,
    pub total_locked: Uint128,
}

pub const LOCKUP_ADDR: Item<Addr> = Item::new("lockup_contract");

pub const PROGRAMS: Map<u64, Program> = Map::new("programs");
pub const PROGRAMS_ID: Item<u64> = Item::new("programs_id");

pub const LAST_EPOCH_PROCESSED: Map<u64, u64> = Map::new("last_epoch_processed");
pub const EPOCH_INFO: Map<(u64, u64), EpochInfo> = Map::new("epoch_info");

pub const WITHDRAWALS: Map<(u64, Addr), u64> = Map::new("withdraw_snapshots");

// keeps track of funding information of various programs
pub const FUNDING_ID: Item<u64> = Item::new("funding_id");

pub struct FundingIndexes<'a> {
    pub(crate) pay_from_epoch: MultiIndex<'a, (u64, u64), Funding, u64>,
}

impl IndexList<Funding> for FundingIndexes<'_> {
    fn get_indexes(
        &'_ self,
    ) -> Box<dyn Iterator<Item = &'_ dyn Index<Funding>> + '_> {
        let v: Vec<&dyn Index<Funding>> = vec![&self.pay_from_epoch];
        Box::new(v.into_iter())
    }
}

pub fn funding<'a>() -> IndexedMap<'a, u64, Funding, FundingIndexes<'a>> {
    let indexes = FundingIndexes {
        pay_from_epoch: MultiIndex::new(
            |_bz, funding: &Funding| -> (_, _) {
                (funding.program_id, funding.pay_from_epoch)
            },
            "funding",
            "pay_from_epoch",
        ),
    };

    IndexedMap::new("funding", indexes)
}
