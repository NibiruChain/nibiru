package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// itemKey is a constant byte key which maps an Item object.
var itemKey = []byte{0x0}

// NewItem instantiates a new Item instance.
func NewItem[V any, PV interface {
	*V
	Object
}](cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8) Item[V, PV] {
	return Item[V, PV]{
		prefix:   []byte{prefix},
		sk:       sk,
		cdc:      newStoreCodec(cdc),
		typeName: typeName(PV(new(V))),
	}
}

// Item represents a state object which will always have one instance
// of itself saved in the namespace.
// Examples are:
//   - config
//   - parameters
//   - a sequence
type Item[V any, PV interface {
	*V
	Object
}] struct {
	_        V
	prefix   []byte
	sk       sdk.StoreKey
	cdc      storeCodec
	typeName string
}

func (i Item[V, PV]) getStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(i.sk), i.prefix)
}

// Get gets the item V or returns an error.
func (i Item[V, PV]) Get(ctx sdk.Context) (V, error) {
	s := i.getStore(ctx)
	bytes := s.Get(itemKey)

	var v V
	if bytes == nil {
		return v, notFoundError(i.typeName, "item")
	}

	i.cdc.unmarshal(bytes, PV(&v))
	return v, nil
}

// GetOr either returns the provided default
// if it's not present in state, or the value found in state.
func (i Item[V, PV]) GetOr(ctx sdk.Context, def V) V {
	got, err := i.Get(ctx)
	if err != nil {
		return def
	}
	return got
}

// Set sets the item value to v.
func (i Item[V, PV]) Set(ctx sdk.Context, v V) {
	i.getStore(ctx).Set(itemKey, i.cdc.marshal(PV(&v)))
}
