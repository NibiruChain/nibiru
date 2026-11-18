//! Write overlay over read-only CosmWasm storage.
//!
//! [`StorageTransaction`] implements [`Storage`] on top of a `&dyn Storage`:
//!
//! - **Reads** check the overlay's pending changes first, then the base
//!   storage. A pending delete hides the base value.
//! - **Writes and deletes** stay in memory in the overlay.
//! - **Range iteration** merges pending changes over the base in key order, in
//!   either direction. `cw-storage-plus` types (`Item`, `Map`, `IndexedMap`,
//!   prefixes and secondary indexes) therefore see the same view they would
//!   see if the changes had been written.
//!
//! The base storage changes only if the caller commits with
//! `txn.prepare().commit(&mut base)`. Dropping the transaction discards every
//! pending change.
//!
//! ## Simulating execution in a query
//!
//! A smart query receives read-only storage, but execute handlers need
//! `DepsMut`. Wrapping the query's storage in a transaction lets a query run
//! an execute handler unchanged and return its `Response`. Dropping the
//! transaction afterward persists nothing:
//!
//! ```
//! use cosmwasm_std::{
//!     testing::mock_dependencies, Deps, DepsMut, Response, StdResult,
//!     Storage,
//! };
//! use nibiru_std::storage_txn::StorageTransaction;
//!
//! /// Stand-in for a contract's execute handler.
//! fn increment(deps: DepsMut) -> StdResult<Response> {
//!     let n = deps.storage.get(b"count").map_or(0, |v| v[0]);
//!     deps.storage.set(b"count", &[n + 1]);
//!     Ok(Response::new().add_attribute("count", (n + 1).to_string()))
//! }
//!
//! /// Body of a smart query that previews `increment`.
//! fn preview(deps: Deps) -> StdResult<Response> {
//!     let mut txn = StorageTransaction::new(deps.storage);
//!     let deps_mut = DepsMut {
//!         storage: &mut txn,
//!         api: deps.api,
//!         querier: deps.querier,
//!     };
//!     increment(deps_mut)
//!     // `txn` drops here without `prepare`, so nothing is committed.
//! }
//!
//! let mut owned = mock_dependencies();
//! owned.storage.set(b"count", &[1]);
//!
//! let res = preview(owned.as_ref())?;
//! assert_eq!(res.attributes[0].value, "2");
//! assert_eq!(owned.storage.get(b"count"), Some(vec![1]));
//! # Ok::<(), cosmwasm_std::StdError>(())
//! ```
//!
//! Do not call [`StorageTransaction::prepare`] in a query. The host rejects
//! writes during a query, but a simulation should not depend on that.
//!
//! What a simulation does **not** reproduce: messages in the returned
//! `Response` (bank sends, calls to other contracts) are not dispatched, and
//! `MessageInfo::funds` is not backed by a real bank transfer. Callers should
//! document both limits wherever they expose a simulated result.
//!
//! ## Costs
//!
//! - Every pending change is held in memory until the transaction drops, so
//!   memory grows with the number of distinct keys written.
//! - Reads that miss the overlay go to the base storage, so they cost the same
//!   host gas as ordinary reads.
//! - A range call walks the overlay's pending keys in that range alongside the
//!   base iterator. Pending deletes are skipped iteratively, without
//!   recursion.
//!
//! ## Origin
//!
//! Ported from the private `transactions` module of
//! [cw-multi-test](https://github.com/CosmWasm/cw-multi-test) v1.2.0
//! (`src/transactions.rs`), Copyright Confio GmbH, licensed under the Apache
//! License, Version 2.0 (<http://www.apache.org/licenses/LICENSE-2.0>).
//! cw-multi-test runs every simulated execute inside the same structure.
//! Changes from the original:
//!
//! - [`transactional`] takes a caller-chosen error type instead of
//!   `anyhow::Result`.
//! - The merged iterator skips consecutive pending deletes in a loop instead
//!   of recursing, so a long run of deletes cannot exhaust the Wasm stack.
//! - Added documentation and tests with `cw-storage-plus` types.

use std::{
    cmp::Ordering,
    collections::BTreeMap,
    iter::{self, Peekable},
    ops::{Bound, RangeBounds},
};

use cosmwasm_std::{Order, Record, Storage};

/// Key-value pair reference returned by `BTreeMap<Vec<u8>, T>::range`.
type BTreeMapPairRef<'a, T = Vec<u8>> = (&'a Vec<u8>, &'a T);

/// Runs `action` against a transaction over `base` and commits its changes
/// only if `action` returns `Ok`. On `Err`, `base` is unchanged, which gives
/// all-or-nothing semantics for a group of writes.
///
/// The closure receives two views: the transaction, which sees pending
/// changes, and `base` as it was before the transaction, which does not.
///
/// ```
/// use cosmwasm_std::{testing::MockStorage, Storage};
/// use nibiru_std::storage_txn::transactional;
///
/// let mut base = MockStorage::new();
/// let res: Result<(), &str> = transactional(&mut base, |txn, _before| {
///     txn.set(b"a", b"1");
///     Err("abort")
/// });
/// assert!(res.is_err());
/// assert_eq!(base.get(b"a"), None); // rolled back
/// ```
pub fn transactional<F, T, E>(base: &mut dyn Storage, action: F) -> Result<T, E>
where
    F: FnOnce(&mut dyn Storage, &dyn Storage) -> Result<T, E>,
{
    let mut cache = StorageTransaction::new(base);
    let res = action(&mut cache, base)?;
    cache.prepare().commit(base);
    Ok(res)
}

/// Storage that buffers writes and deletes over read-only base storage.
///
/// See the [module documentation](self) for read, write and iteration
/// semantics. A transaction can wrap another transaction to nest them; each
/// level commits into the one below it.
pub struct StorageTransaction<'a> {
    /// Read-only access to the backing storage.
    storage: &'a dyn Storage,
    /// Pending changes, by key, not flushed to the backing storage.
    local_state: BTreeMap<Vec<u8>, Delta>,
    /// Ordered log of pending changes, replayed on commit.
    rep_log: RepLog,
}

impl<'a> StorageTransaction<'a> {
    /// Creates an empty transaction over `storage`. The transaction borrows
    /// `storage` for its lifetime and never writes to it.
    pub fn new(storage: &'a dyn Storage) -> Self {
        StorageTransaction {
            storage,
            local_state: BTreeMap::new(),
            rep_log: RepLog::new(),
        }
    }

    /// Consumes the transaction and returns its change log. Pass the log to
    /// [`RepLog::commit`] to apply it, once the borrow of the base storage
    /// has ended.
    pub fn prepare(self) -> RepLog {
        self.rep_log
    }
}

impl<'a> Storage for StorageTransaction<'a> {
    /// Returns the pending value for `key`, `None` if `key` has a pending
    /// delete, or otherwise the base storage value.
    fn get(&self, key: &[u8]) -> Option<Vec<u8>> {
        match self.local_state.get(key) {
            Some(Delta::Set { value }) => Some(value.clone()),
            Some(Delta::Delete {}) => None,
            None => self.storage.get(key),
        }
    }

    /// Iterates records with keys in `[start, end)`, in `order`. A pending
    /// value replaces the base record with the same key, and a pending
    /// delete hides it. `start > end` yields nothing.
    fn range<'b>(
        &'b self,
        start: Option<&[u8]>,
        end: Option<&[u8]>,
        order: Order,
    ) -> Box<dyn Iterator<Item = Record> + 'b> {
        let bounds = range_bounds(start, end);

        // `BTreeMap::range` panics when start > end. That case is an empty
        // range.
        let local: Box<dyn Iterator<Item = BTreeMapPairRef<Delta>>> =
            match (bounds.start_bound(), bounds.end_bound()) {
                (Bound::Included(start), Bound::Excluded(end))
                    if start > end =>
                {
                    Box::new(iter::empty())
                }
                _ => {
                    let local_raw = self.local_state.range(bounds);
                    match order {
                        Order::Ascending => Box::new(local_raw),
                        Order::Descending => Box::new(local_raw.rev()),
                    }
                }
            };

        let base = self.storage.range(start, end, order);
        Box::new(MergeOverlay::new(local, base, order))
    }

    /// Records a pending write. The base storage is unchanged.
    fn set(&mut self, key: &[u8], value: &[u8]) {
        let op = Op::Set {
            key: key.to_vec(),
            value: value.to_vec(),
        };
        self.local_state.insert(key.to_vec(), op.to_delta());
        self.rep_log.append(op);
    }

    /// Records a pending delete. The base storage is unchanged.
    fn remove(&mut self, key: &[u8]) {
        let op = Op::Delete { key: key.to_vec() };
        self.local_state.insert(key.to_vec(), op.to_delta());
        self.rep_log.append(op);
    }
}

/// Ordered changes from a [`StorageTransaction`], returned by
/// [`StorageTransaction::prepare`]. Dropping a `RepLog` without calling
/// [`RepLog::commit`] discards the changes.
pub struct RepLog {
    ops_log: Vec<Op>,
}

impl RepLog {
    fn new() -> Self {
        RepLog { ops_log: vec![] }
    }

    fn append(&mut self, op: Op) {
        self.ops_log.push(op);
    }

    /// Applies every logged write and delete to `storage` in the order they
    /// were made, so a later change to a key overrides an earlier one.
    pub fn commit(self, storage: &mut dyn Storage) {
        for op in self.ops_log {
            op.apply(storage);
        }
    }
}

/// One logged storage operation, replayed by [`RepLog::commit`].
enum Op {
    Set { key: Vec<u8>, value: Vec<u8> },
    Delete { key: Vec<u8> },
}

impl Op {
    fn apply(&self, storage: &mut dyn Storage) {
        match self {
            Op::Set { key, value } => storage.set(key, value),
            Op::Delete { key } => storage.remove(key),
        }
    }

    fn to_delta(&self) -> Delta {
        match self {
            Op::Set { value, .. } => Delta::Set {
                value: value.clone(),
            },
            Op::Delete { .. } => Delta::Delete {},
        }
    }
}

/// Pending change for one key: the value side of `local_state`.
enum Delta {
    Set { value: Vec<u8> },
    Delete {},
}

/// Merges pending changes (`left`) over base records (`right`) in `order`.
/// On equal keys the pending change wins; pending deletes are skipped.
struct MergeOverlay<'a, L, R>
where
    L: Iterator<Item = BTreeMapPairRef<'a, Delta>>,
    R: Iterator<Item = Record>,
{
    left: Peekable<L>,
    right: Peekable<R>,
    order: Order,
}

impl<'a, L, R> MergeOverlay<'a, L, R>
where
    L: Iterator<Item = BTreeMapPairRef<'a, Delta>>,
    R: Iterator<Item = Record>,
{
    fn new(left: L, right: R, order: Order) -> Self {
        MergeOverlay {
            left: left.peekable(),
            right: right.peekable(),
            order,
        }
    }

    /// Which side supplies the next record, given the next key on each side.
    fn pick(&mut self) -> Pick {
        let order = self.order;
        match (self.left.peek(), self.right.peek()) {
            (Some((lkey, _)), Some((rkey, _))) => {
                let cmp = match order {
                    Order::Ascending => lkey.as_slice().cmp(rkey),
                    Order::Descending => rkey.as_slice().cmp(lkey),
                };
                match cmp {
                    Ordering::Less => Pick::Left,
                    Ordering::Equal => Pick::LeftShadowingRight,
                    Ordering::Greater => Pick::Right,
                }
            }
            (Some(_), None) => Pick::Left,
            (None, Some(_)) => Pick::Right,
            (None, None) => Pick::Done,
        }
    }
}

/// Outcome of comparing the next pending key with the next base key.
enum Pick {
    Left,
    /// Equal keys: the pending change replaces the base record.
    LeftShadowingRight,
    Right,
    Done,
}

impl<'a, L, R> Iterator for MergeOverlay<'a, L, R>
where
    L: Iterator<Item = BTreeMapPairRef<'a, Delta>>,
    R: Iterator<Item = Record>,
{
    type Item = Record;

    fn next(&mut self) -> Option<Self::Item> {
        loop {
            match self.pick() {
                Pick::Done => return None,
                Pick::Right => return self.right.next(),
                Pick::LeftShadowingRight => {
                    let _ = self.right.next();
                }
                Pick::Left => {}
            }
            let (lkey, lval) = self.left.next()?;
            match lval {
                Delta::Set { value } => {
                    return Some((lkey.clone(), value.clone()))
                }
                Delta::Delete {} => continue,
            }
        }
    }
}

/// Converts `Storage::range` arguments into `BTreeMap` bounds: inclusive
/// start, exclusive end, either side unbounded when `None`.
fn range_bounds(
    start: Option<&[u8]>,
    end: Option<&[u8]>,
) -> impl RangeBounds<Vec<u8>> {
    (
        start.map_or(Bound::Unbounded, |x| Bound::Included(x.to_vec())),
        end.map_or(Bound::Unbounded, |x| Bound::Excluded(x.to_vec())),
    )
}

#[cfg(test)]
mod test {
    use super::*;
    use std::cell::RefCell;
    use std::ops::{Deref, DerefMut};

    use cosmwasm_std::MemoryStorage;

    #[test]
    fn wrap_storage() {
        let mut store = MemoryStorage::new();
        let mut wrap = StorageTransaction::new(&store);
        wrap.set(b"foo", b"bar");

        assert_eq!(None, store.get(b"foo"));
        wrap.prepare().commit(&mut store);
        assert_eq!(Some(b"bar".to_vec()), store.get(b"foo"));
    }

    #[test]
    fn wrap_ref_cell() {
        let store = RefCell::new(MemoryStorage::new());
        let ops = {
            let refer = store.borrow();
            let mut wrap = StorageTransaction::new(refer.deref());
            wrap.set(b"foo", b"bar");
            assert_eq!(None, store.borrow().get(b"foo"));
            wrap.prepare()
        };
        ops.commit(store.borrow_mut().deref_mut());
        assert_eq!(Some(b"bar".to_vec()), store.borrow().get(b"foo"));
    }

    #[test]
    fn wrap_box_storage() {
        let mut store: Box<MemoryStorage> = Box::new(MemoryStorage::new());
        let mut wrap = StorageTransaction::new(store.as_ref());
        wrap.set(b"foo", b"bar");

        assert_eq!(None, store.get(b"foo"));
        wrap.prepare().commit(store.as_mut());
        assert_eq!(Some(b"bar".to_vec()), store.get(b"foo"));
    }

    #[test]
    fn wrap_box_dyn_storage() {
        let mut store: Box<dyn Storage> = Box::new(MemoryStorage::new());
        let mut wrap = StorageTransaction::new(store.as_ref());
        wrap.set(b"foo", b"bar");

        assert_eq!(None, store.get(b"foo"));
        wrap.prepare().commit(store.as_mut());
        assert_eq!(Some(b"bar".to_vec()), store.get(b"foo"));
    }

    #[test]
    fn wrap_ref_cell_dyn_storage() {
        let inner: Box<dyn Storage> = Box::new(MemoryStorage::new());
        let store = RefCell::new(inner);
        // The `Ref` must outlive the transaction, and `commit` must wait until
        // the shared borrow ends.
        let ops = {
            let refer = store.borrow();
            let mut wrap = StorageTransaction::new(refer.as_ref());
            wrap.set(b"foo", b"bar");

            assert_eq!(None, store.borrow().get(b"foo"));
            wrap.prepare()
        };
        ops.commit(store.borrow_mut().as_mut());
        assert_eq!(Some(b"bar".to_vec()), store.borrow().get(b"foo"));
    }

    /// Adds data to `store` and checks iteration. `store` must start with
    /// exactly one key, "foo" = "bar".
    fn iterator_test_suite<S: Storage>(store: &mut S) {
        assert_eq!(store.get(b"foo"), Some(b"bar".to_vec()));
        assert_eq!(store.range(None, None, Order::Ascending).count(), 1);

        store.set(b"ant", b"hill");
        store.set(b"ze", b"bra");

        // Noise that iteration must ignore.
        store.set(b"bye", b"bye");
        store.remove(b"bye");

        let collect = |s: Option<&[u8]>, e: Option<&[u8]>, o: Order| {
            store.range(s, e, o).collect::<Vec<Record>>()
        };
        let rec = |k: &[u8], v: &[u8]| (k.to_vec(), v.to_vec());

        // unbounded
        assert_eq!(
            collect(None, None, Order::Ascending),
            vec![
                rec(b"ant", b"hill"),
                rec(b"foo", b"bar"),
                rec(b"ze", b"bra")
            ]
        );
        assert_eq!(
            collect(None, None, Order::Descending),
            vec![
                rec(b"ze", b"bra"),
                rec(b"foo", b"bar"),
                rec(b"ant", b"hill")
            ]
        );
        // bounded
        assert_eq!(
            collect(Some(b"f"), Some(b"n"), Order::Ascending),
            vec![rec(b"foo", b"bar")]
        );
        assert_eq!(
            collect(Some(b"air"), Some(b"loop"), Order::Descending),
            vec![rec(b"foo", b"bar"), rec(b"ant", b"hill")]
        );
        // bounded empty [a, a)
        assert_eq!(collect(Some(b"foo"), Some(b"foo"), Order::Ascending), []);
        assert_eq!(collect(Some(b"foo"), Some(b"foo"), Order::Descending), []);
        // bounded empty [a, b) with b < a
        assert_eq!(collect(Some(b"z"), Some(b"a"), Order::Ascending), []);
        assert_eq!(collect(Some(b"z"), Some(b"a"), Order::Descending), []);
        // right unbounded
        assert_eq!(
            collect(Some(b"f"), None, Order::Ascending),
            vec![rec(b"foo", b"bar"), rec(b"ze", b"bra")]
        );
        assert_eq!(
            collect(Some(b"f"), None, Order::Descending),
            vec![rec(b"ze", b"bra"), rec(b"foo", b"bar")]
        );
        // left unbounded
        assert_eq!(
            collect(None, Some(b"f"), Order::Ascending),
            vec![rec(b"ant", b"hill")]
        );
        assert_eq!(
            collect(None, Some(b"no"), Order::Descending),
            vec![rec(b"foo", b"bar"), rec(b"ant", b"hill")]
        );
    }

    #[test]
    fn delete_local() {
        let mut base = Box::new(MemoryStorage::new());
        let mut check = StorageTransaction::new(base.as_ref());
        check.set(b"foo", b"bar");
        check.set(b"food", b"bank");
        check.remove(b"foo");

        assert_eq!(check.get(b"foo"), None);
        assert_eq!(check.get(b"food"), Some(b"bank".to_vec()));

        check.prepare().commit(base.as_mut());
        assert_eq!(base.get(b"foo"), None);
        assert_eq!(base.get(b"food"), Some(b"bank".to_vec()));
    }

    #[test]
    fn delete_from_base() {
        let mut base = Box::new(MemoryStorage::new());
        base.set(b"foo", b"bar");
        let mut check = StorageTransaction::new(base.as_ref());
        check.set(b"food", b"bank");
        check.remove(b"foo");

        assert_eq!(check.get(b"foo"), None);
        assert_eq!(check.get(b"food"), Some(b"bank".to_vec()));

        check.prepare().commit(base.as_mut());
        assert_eq!(base.get(b"foo"), None);
        assert_eq!(base.get(b"food"), Some(b"bank".to_vec()));
    }

    #[test]
    fn storage_transaction_iterator_empty_base() {
        let base = MemoryStorage::new();
        let mut check = StorageTransaction::new(&base);
        check.set(b"foo", b"bar");
        iterator_test_suite(&mut check);
    }

    #[test]
    fn storage_transaction_iterator_with_base_data() {
        let mut base = MemoryStorage::new();
        base.set(b"foo", b"bar");
        let mut check = StorageTransaction::new(&base);
        iterator_test_suite(&mut check);
    }

    #[test]
    fn storage_transaction_iterator_removed_items_from_base() {
        let mut base = Box::new(MemoryStorage::new());
        base.set(b"foo", b"bar");
        base.set(b"food", b"bank");
        let mut check = StorageTransaction::new(base.as_ref());
        check.remove(b"food");
        iterator_test_suite(&mut check);
    }

    #[test]
    fn commit_writes_through() {
        let mut base = Box::new(MemoryStorage::new());
        base.set(b"foo", b"bar");

        let mut check = StorageTransaction::new(base.as_ref());
        assert_eq!(check.get(b"foo"), Some(b"bar".to_vec()));
        check.set(b"subtx", b"works");
        check.prepare().commit(base.as_mut());

        assert_eq!(base.get(b"subtx"), Some(b"works".to_vec()));
    }

    #[test]
    fn storage_remains_readable() {
        let mut base = MemoryStorage::new();
        base.set(b"foo", b"bar");

        let mut stx1 = StorageTransaction::new(&base);
        assert_eq!(stx1.get(b"foo"), Some(b"bar".to_vec()));

        stx1.set(b"subtx", b"works");
        assert_eq!(stx1.get(b"subtx"), Some(b"works".to_vec()));

        // The base is unchanged until commit.
        assert_eq!(base.get(b"subtx"), None);

        stx1.prepare().commit(&mut base);
        assert_eq!(base.get(b"subtx"), Some(b"works".to_vec()));
    }

    #[test]
    fn ignore_same_as_rollback() {
        let mut base = MemoryStorage::new();
        base.set(b"foo", b"bar");

        let mut check = StorageTransaction::new(&base);
        assert_eq!(check.get(b"foo"), Some(b"bar".to_vec()));
        check.set(b"subtx", b"works");

        assert_eq!(base.get(b"subtx"), None);
    }

    // ---- Additions beyond the upstream suite ----

    fn snapshot(store: &dyn Storage) -> Vec<Record> {
        store.range(None, None, Order::Ascending).collect()
    }

    #[test]
    fn drop_without_commit_leaves_base_identical() {
        let mut base = MemoryStorage::new();
        for i in 0u8..50 {
            base.set(&[b'k', i], &[i]);
        }
        let before = snapshot(&base);
        {
            let mut txn = StorageTransaction::new(&base);
            for i in 0u8..50 {
                if i % 2 == 0 {
                    txn.remove(&[b'k', i]);
                } else {
                    txn.set(&[b'k', i], b"changed");
                }
            }
            txn.set(b"new", b"key");
            assert_eq!(snapshot(&txn).len(), 26);
        }
        assert_eq!(snapshot(&base), before);
    }

    #[test]
    fn long_run_of_deletes_is_skipped_without_recursion() {
        let mut base = MemoryStorage::new();
        let n: u32 = 20_000;
        for i in 0..n {
            base.set(&i.to_be_bytes(), b"v");
        }
        base.set(b"\xff\xff\xff\xff\xff", b"last");
        let mut txn = StorageTransaction::new(&base);
        for i in 0..n {
            txn.remove(&i.to_be_bytes());
        }
        assert_eq!(
            snapshot(&txn),
            vec![(b"\xff\xff\xff\xff\xff".to_vec(), b"last".to_vec())]
        );
        let desc: Vec<Record> =
            txn.range(None, None, Order::Descending).collect();
        assert_eq!(desc.len(), 1);
    }

    #[test]
    fn delete_then_reinsert_and_overwrite() {
        let mut base = MemoryStorage::new();
        base.set(b"a", b"1");
        base.set(b"b", b"2");
        let mut txn = StorageTransaction::new(&base);
        txn.remove(b"a");
        txn.set(b"a", b"3");
        txn.set(b"b", b"4");
        txn.remove(b"b");
        assert_eq!(snapshot(&txn), vec![(b"a".to_vec(), b"3".to_vec())]);

        txn.prepare().commit(&mut base);
        assert_eq!(snapshot(&base), vec![(b"a".to_vec(), b"3".to_vec())]);
    }

    #[test]
    fn transactional_commits_on_ok_and_discards_on_err() {
        let mut base = MemoryStorage::new();
        let ok: Result<(), String> = transactional(&mut base, |txn, _| {
            txn.set(b"ok", b"1");
            Ok(())
        });
        assert!(ok.is_ok());
        assert_eq!(base.get(b"ok"), Some(b"1".to_vec()));

        let err: Result<(), String> = transactional(&mut base, |txn, read| {
            txn.set(b"err", b"1");
            assert_eq!(read.get(b"err"), None);
            Err("boom".to_string())
        });
        assert_eq!(err, Err("boom".to_string()));
        assert_eq!(base.get(b"err"), None);
    }

    mod storage_plus {
        use super::*;
        use cosmwasm_std::testing::MockStorage;
        use cw_storage_plus::{
            Bound as SpBound, Index, IndexList, IndexedMap, Item, Map,
            MultiIndex,
        };
        use serde::{Deserialize, Serialize};

        #[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq)]
        struct Trade {
            market: u32,
            size: u64,
        }

        struct TradeIndexes<'a> {
            market: MultiIndex<'a, u32, Trade, (&'a str, u64)>,
        }

        impl<'a> IndexList<Trade> for TradeIndexes<'a> {
            fn get_indexes(
                &'_ self,
            ) -> Box<dyn Iterator<Item = &'_ dyn Index<Trade>> + '_>
            {
                let v: Vec<&dyn Index<Trade>> = vec![&self.market];
                Box::new(v.into_iter())
            }
        }

        fn trades<'a>() -> IndexedMap<'a, (&'a str, u64), Trade, TradeIndexes<'a>>
        {
            IndexedMap::new(
                "trades",
                TradeIndexes {
                    market: MultiIndex::new(
                        |_pk, t: &Trade| t.market,
                        "trades",
                        "trades__market",
                    ),
                },
            )
        }

        const CONFIG: Item<u64> = Item::new("config");
        const TOTALS: Map<u32, u64> = Map::new("totals");

        fn seeded() -> MockStorage {
            let mut base = MockStorage::new();
            CONFIG.save(&mut base, &7).unwrap();
            for (user, idx, market, size) in [
                ("alice", 0, 1, 10),
                ("alice", 1, 2, 20),
                ("bob", 0, 1, 30),
                ("carol", 0, 1, 40),
            ] {
                trades()
                    .save(&mut base, (user, idx), &Trade { market, size })
                    .unwrap();
            }
            TOTALS.save(&mut base, 1, &80).unwrap();
            TOTALS.save(&mut base, 2, &20).unwrap();
            base
        }

        fn market_keys(
            store: &dyn Storage,
            market: u32,
            order: Order,
        ) -> Vec<(String, u64)> {
            trades()
                .idx
                .market
                .prefix(market)
                .keys(store, None, None, order)
                .map(|k| {
                    let (u, i) = k.unwrap();
                    (u, i)
                })
                .collect()
        }

        #[test]
        fn item_and_map_reads_see_pending_writes() {
            let base = seeded();
            let mut txn = StorageTransaction::new(&base);
            CONFIG.save(&mut txn, &8).unwrap();
            TOTALS.remove(&mut txn, 2);
            TOTALS.save(&mut txn, 3, &5).unwrap();

            assert_eq!(CONFIG.load(&txn).unwrap(), 8);
            let totals: Vec<(u32, u64)> = TOTALS
                .range(&txn, None, None, Order::Ascending)
                .map(Result::unwrap)
                .collect();
            assert_eq!(totals, vec![(1, 80), (3, 5)]);
            let desc: Vec<u32> = TOTALS
                .keys(&txn, None, None, Order::Descending)
                .map(Result::unwrap)
                .collect();
            assert_eq!(desc, vec![3, 1]);

            assert_eq!(CONFIG.load(&base).unwrap(), 7);
            assert_eq!(TOTALS.load(&base, 2).unwrap(), 20);
        }

        #[test]
        fn indexed_map_multi_index_follows_pending_changes() {
            /// Moves bob to market 2, closes carol, and opens dave on market 1.
            fn mutate(store: &mut dyn Storage) {
                let t = |market, size| Trade { market, size };
                trades().save(store, ("bob", 0), &t(2, 30)).unwrap();
                trades().remove(store, ("carol", 0)).unwrap();
                trades().save(store, ("dave", 0), &t(1, 50)).unwrap();
            }

            // Oracle: the same writes applied directly to plain storage.
            let mut expected = seeded();
            mutate(&mut expected);

            let base = seeded();
            let before = snapshot(&base);
            let mut txn = StorageTransaction::new(&base);
            mutate(&mut txn);

            for order in [Order::Ascending, Order::Descending] {
                for market in [1, 2] {
                    assert_eq!(
                        market_keys(&txn, market, order),
                        market_keys(&expected, market, order),
                        "market {market} ascending={}",
                        matches!(order, Order::Ascending)
                    );
                }
            }
            assert_eq!(snapshot(&txn), snapshot(&expected));

            let mut m1 = market_keys(&txn, 1, Order::Ascending);
            m1.sort();
            assert_eq!(
                m1,
                vec![("alice".to_string(), 0), ("dave".to_string(), 0)]
            );
            assert_eq!(market_keys(&base, 1, Order::Ascending).len(), 3);
            drop(txn);
            assert_eq!(snapshot(&base), before);
        }

        #[test]
        fn prefix_range_with_bounds_merges_both_sources() {
            let base = seeded();
            let mut txn = StorageTransaction::new(&base);
            trades()
                .save(&mut txn, ("alice", 2), &Trade { market: 1, size: 1 })
                .unwrap();
            trades().remove(&mut txn, ("alice", 0)).unwrap();

            let alice: Vec<u64> = trades()
                .prefix("alice")
                .keys(
                    &txn,
                    Some(SpBound::inclusive(1u64)),
                    None,
                    Order::Ascending,
                )
                .map(Result::unwrap)
                .collect();
            assert_eq!(alice, vec![1, 2]);

            let alice_desc: Vec<u64> = trades()
                .prefix("alice")
                .keys(&txn, None, None, Order::Descending)
                .map(Result::unwrap)
                .collect();
            assert_eq!(alice_desc, vec![2, 1]);
        }
    }
}
