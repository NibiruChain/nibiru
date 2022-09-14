package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
)

// KeySet wraps the default Map, but is used only for
// keys.Key presence and ranging functionalities.
type KeySet[K keys.Key] Map[K, setObject, *setObject]

// KeySetIterator wraps the default MapIterator, but is used only
// for keys.Key ranging.
type KeySetIterator[K keys.Key] MapIterator[K, setObject, *setObject]

// NewKeySet instantiates a new KeySet.
func NewKeySet[K keys.Key](cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8) KeySet[K] {
	return KeySet[K]{
		cdc:      newStoreCodec(cdc),
		sk:       sk,
		prefix:   []byte{prefix},
		typeName: typeName(new(setObject)),
	}
}

// Has reports whether the key K is present or not in the set.
func (s KeySet[K]) Has(ctx sdk.Context, k K) bool {
	_, err := (Map[K, setObject, *setObject])(s).Get(ctx, k)
	return err == nil
}

// Insert inserts the key K in the set.
func (s KeySet[K]) Insert(ctx sdk.Context, k K) {
	(Map[K, setObject, *setObject])(s).Insert(ctx, k, setObject{})
}

// Delete deletes the key from the set.
// Does not check if the key exists or not.
func (s KeySet[K]) Delete(ctx sdk.Context, k K) {
	_ = (Map[K, setObject, *setObject])(s).Delete(ctx, k)
}

// Iterate returns a KeySetIterator over the provided keys.Range of keys.
func (s KeySet[K]) Iterate(ctx sdk.Context, r keys.Range[K]) KeySetIterator[K] {
	mi := (Map[K, setObject, *setObject])(s).Iterate(ctx, r)
	return (KeySetIterator[K])(mi)
}

// Close closes the KeySetIterator.
// No other operation is valid.
func (s KeySetIterator[K]) Close() {
	(MapIterator[K, setObject, *setObject])(s).Close()
}

// Next moves the iterator onto the next key.
func (s KeySetIterator[K]) Next() {
	(MapIterator[K, setObject, *setObject])(s).Next()
}

// Valid checks if the iterator is still valid.
func (s KeySetIterator[K]) Valid() bool {
	return (MapIterator[K, setObject, *setObject])(s).Valid()
}

// Key returns the current iterator key.
func (s KeySetIterator[K]) Key() K {
	return (MapIterator[K, setObject, *setObject])(s).Key()
}

// Keys consumes the iterator fully and returns all the available keys.
// The KeySetIterator is closed after this operation.
func (s KeySetIterator[K]) Keys() []K {
	return (MapIterator[K, setObject, *setObject])(s).Keys()
}
