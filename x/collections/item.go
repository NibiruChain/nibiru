package collections

import (
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// itemKey is a constant byte key which maps an Item object.
const itemKey uint64 = 0

// NewItem instantiates a new Item instance.
func NewItem[V any](sk types.StoreKey, namespace Namespace, valueEncoder ValueEncoder[V]) Item[V] {
	return (Item[V])(NewMap[uint64, V](sk, namespace, uint64Key{}, valueEncoder))
}

// Item represents a state object which will always have one instance
// of itself saved in the namespace.
// Examples are:
//   - config
//   - parameters
//   - a sequence
//
// It builds on top of a Map with a constant key.
type Item[V any] Map[uint64, V]

// Get gets the item V or returns an error.
func (i Item[V]) Get(ctx sdk.Context) (V, error) { return (Map[uint64, V])(i).Get(ctx, itemKey) }

// GetOr either returns the provided default
// if it's not present in state, or the value found in state.
func (i Item[V]) GetOr(ctx sdk.Context, def V) V { return (Map[uint64, V])(i).GetOr(ctx, itemKey, def) }

// Set sets the item value to v.
func (i Item[V]) Set(ctx sdk.Context, v V) { (Map[uint64, V])(i).Insert(ctx, itemKey, v) }

// NewItem instantiates a new Item instance.
func NewItemTransient[V any](
	sk types.StoreKey, namespace Namespace, valueEncoder ValueEncoder[V],
) ItemTransient[V] {
	return (ItemTransient[V])(NewMapTransient[uint64, V](sk, namespace, uint64Key{}, valueEncoder))
}

// ItemTransient: An [Item] that maps to a transient key-value store (KV store)
// instead of a persistent one. A Transient KV Store
// is used for data that does not need to persist beyond the execution of the
// current block or transaction.
//
// This can include temporary calculations, intermediate state data in
// transactions or ephemeral data used in block processing. Data is a transient
// store is cleared after the block is processed.
//
// Transient KV stores have markedly lower costs for all operations (10% of the
// persistent cost) and a read cost per byte of zero.
type ItemTransient[V any] MapTransient[uint64, V]

func (i ItemTransient[V]) Get(ctx sdk.Context) (V, error) {
	return (MapTransient[uint64, V])(i).Get(ctx, itemKey)
}

// GetOr either returns the provided default
// if it's not present in state, or the value found in state.
func (i ItemTransient[V]) GetOr(ctx sdk.Context, def V) V {
	return (MapTransient[uint64, V])(i).GetOr(ctx, itemKey, def)
}

// Set sets the item value to v.
func (i ItemTransient[V]) Set(ctx sdk.Context, v V) {
	(MapTransient[uint64, V])(i).Insert(ctx, itemKey, v)
}
