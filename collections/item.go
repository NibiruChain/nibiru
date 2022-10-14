package collections

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// itemKey is a constant byte key which maps an Item object.
var itemKey uint64 = 0

// NewItem instantiates a new Item instance.
func NewItem[V any](sk sdk.StoreKey, namespace Namespace, valueEncoder ValueEncoder[V]) Item[V] {
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
