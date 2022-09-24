package collections

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
)

// nilObject is used when no object functionality is needed.
// Essentially a null object useful for keysets.
type nilObject struct{}

func (n nilObject) String() string {
	panic("must never be called")
}

func (n nilObject) Marshal() ([]byte, error) {
	return []byte{}, nil
}

func (n nilObject) Unmarshal(b []byte) error {
	if !bytes.Equal(b, []byte{}) {
		panic("bad usage")
	}
	return nil
}

var _ Object = (*nilObject)(nil)

// KeySet wraps the default Map, but is used only for
// keys.Key presence and ranging functionalities.
type KeySet[K keys.Key] Map[K, nilObject, *nilObject]

// KeySetIterator wraps the default MapIterator, but is used only
// for keys.Key ranging.
type KeySetIterator[K keys.Key] MapIterator[K, nilObject, *nilObject]

// NewKeySet instantiates a new KeySet.
func NewKeySet[K keys.Key](cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8) KeySet[K] {
	return KeySet[K]{
		cdc:      newStoreCodec(cdc),
		sk:       sk,
		prefix:   []byte{prefix},
		typeName: typeName(new(nilObject)),
	}
}

// Has reports whether the key K is present or not in the set.
func (s KeySet[K]) Has(ctx sdk.Context, k K) bool {
	_, err := (Map[K, nilObject, *nilObject])(s).Get(ctx, k)
	return err == nil
}

// Insert inserts the key K in the set.
func (s KeySet[K]) Insert(ctx sdk.Context, k K) {
	(Map[K, nilObject, *nilObject])(s).Insert(ctx, k, nilObject{})
}

// Delete deletes the key from the set.
// Does not check if the key exists or not.
func (s KeySet[K]) Delete(ctx sdk.Context, k K) {
	_ = (Map[K, nilObject, *nilObject])(s).Delete(ctx, k)
}

// Iterate returns a KeySetIterator over the provided keys.Range of keys.
func (s KeySet[K]) Iterate(ctx sdk.Context, r keys.Range[K]) KeySetIterator[K] {
	mi := (Map[K, nilObject, *nilObject])(s).Iterate(ctx, r)
	return (KeySetIterator[K])(mi)
}

// Close closes the KeySetIterator.
// No other operation is valid.
func (s KeySetIterator[K]) Close() {
	(MapIterator[K, nilObject, *nilObject])(s).Close()
}

// Next moves the iterator onto the next key.
func (s KeySetIterator[K]) Next() {
	(MapIterator[K, nilObject, *nilObject])(s).Next()
}

// Valid checks if the iterator is still valid.
func (s KeySetIterator[K]) Valid() bool {
	return (MapIterator[K, nilObject, *nilObject])(s).Valid()
}

// Key returns the current iterator key.
func (s KeySetIterator[K]) Key() K {
	return (MapIterator[K, nilObject, *nilObject])(s).Key()
}

// Keys consumes the iterator fully and returns all the available keys.
// The KeySetIterator is closed after this operation.
func (s KeySetIterator[K]) Keys() []K {
	return (MapIterator[K, nilObject, *nilObject])(s).Keys()
}
