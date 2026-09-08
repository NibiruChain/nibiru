use cosmwasm_schema::cw_serde;
use cosmwasm_std::Coin;
use cw_storage_plus::{Index, IndexList, IndexedMap, Item, MultiIndex};

/// A sentinel value to represent that a Lock is not currently unlocking. It
/// ensures that the "not unlocking" state is distinct from any valid block
/// height and allows the contract to check ifi a Lock started unlocking or not.
pub const NOT_UNLOCKING_BLOCK_IDENTIFIER: u64 = u64::MAX;

pub const LOCKS_ID: Item<u64> = Item::new("locks_id");

/// Represents a lock on funds in the contract.
///
/// A `Lock` is created when a user locks up their funds for a specified duration.
/// It keeps track of the locked funds, owner, duration, and various block heights
/// related to the lock's lifecycle.
#[cw_serde]
pub struct Lock {
    /// Unique identifier for the lock.
    pub id: u64,
    /// The amount and denomination of the locked funds.
    pub coin: Coin,
    /// The address of the lock owner.
    pub owner: String,
    /// The duration of the lock in number of blocks.
    pub duration_blocks: u64,
    /// The block height at which the lock was created.
    pub start_block: u64,
    /// The block height at which the lock ends.
    ///
    /// This is set to `NOT_UNLOCKING_BLOCK_IDENTIFIER` when the lock is created,
    /// and updated to an actual block height when unlocking is initiated.
    pub end_block: u64,
    /// Indicates whether the funds have been withdrawn after the lock period.
    pub funds_withdrawn: bool,
}

/// Lock Lifecycle States
#[derive(Debug, PartialEq)]
pub enum LockState {
    /// The lock has been created and funds are locked
    FundedPreUnlock,
    /// Unlocking has been initiated, but the lock period hasn't expired
    Unlocking,
    /// The lock period has expired, funds are ready for withdrawal
    Matured,
    /// Funds have been withdrawn, the lock is now inactive
    Withdrawn,
}

impl Lock {
    /// Computes the lifecycle state of the Lock
    pub fn state(&self, current_block: u64) -> LockState {
        if self.funds_withdrawn {
            return LockState::Withdrawn;
        }

        match self.end_block {
            NOT_UNLOCKING_BLOCK_IDENTIFIER => LockState::FundedPreUnlock,
            end_block if current_block >= end_block => LockState::Matured,
            _ => LockState::Unlocking,
        }
    }
}

pub struct LockIndexes<'a> {
    pub denom_end: MultiIndex<'a, (String, u64), Lock, u64>,
    pub addr_denom_end: MultiIndex<'a, (String, String, u64), Lock, u64>,
    pub denom_start: MultiIndex<'a, (String, u64), Lock, u64>,
    pub addr_denom_start: MultiIndex<'a, (String, String, u64), Lock, u64>,
}

impl<'a> IndexList<Lock> for LockIndexes<'a> {
    fn get_indexes(
        &'_ self,
    ) -> Box<dyn Iterator<Item = &'_ dyn Index<Lock>> + '_> {
        let v: Vec<&dyn Index<Lock>> = vec![
            &self.addr_denom_end,
            &self.denom_end,
            &self.denom_start,
            &self.addr_denom_start,
        ];
        Box::new(v.into_iter())
    }
}

pub fn locks() -> IndexedMap<'static, u64, Lock, LockIndexes<'static>> {
    let indexes = LockIndexes {
        addr_denom_end: MultiIndex::new(
            |_bz, lock: &Lock| -> (_, _, _) {
                (lock.owner.clone(), lock.coin.denom.clone(), lock.end_block)
            },
            "locks",
            "addr_denom_end",
        ),
        denom_start: MultiIndex::new(
            |_bz, lock: &Lock| -> (_, _) {
                (lock.coin.denom.clone(), lock.start_block)
            },
            "locks",
            "denom_start",
        ),

        denom_end: MultiIndex::new(
            |_bz, lock: &Lock| -> (_, _) {
                (lock.coin.denom.clone(), lock.end_block)
            },
            "locks",
            "denom_end",
        ),
        addr_denom_start: MultiIndex::new(
            |_bz, lock: &Lock| -> (_, _, _) {
                (
                    lock.owner.clone(),
                    lock.coin.denom.clone(),
                    lock.start_block,
                )
            },
            "locks",
            "addr_denom_start",
        ),
    };

    IndexedMap::new("locks", indexes)
}
