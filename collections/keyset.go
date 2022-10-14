package collections

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// KeySet wraps the default Map, but is used only for
// keys.Key presence and ranging functionalities.
type KeySet[K any] Map[K, setObject]

// KeySetIterator wraps the default MapIterator, but is used only
// for keys.Key ranging.
type KeySetIterator[K any] Iterator[K, setObject]

// NewKeySet instantiates a new KeySet.
func NewKeySet[K any](sk sdk.StoreKey, namespace Namespace, keyEncoder KeyEncoder[K]) KeySet[K] {
	return (KeySet[K])(NewMap[K, setObject](sk, namespace, keyEncoder, setObject{}))
}

// Has reports whether the key K is present or not in the set.
func (s KeySet[K]) Has(ctx sdk.Context, k K) bool {
	_, err := (Map[K, setObject])(s).Get(ctx, k)
	return err == nil
}

// Insert inserts the key K in the set.
func (s KeySet[K]) Insert(ctx sdk.Context, k K) {
	(Map[K, setObject])(s).Insert(ctx, k, setObject{})
}

// Delete deletes the key from the set.
// Does not check if the key exists or not.
func (s KeySet[K]) Delete(ctx sdk.Context, k K) {
	_ = (Map[K, setObject])(s).Delete(ctx, k)
}

// Iterate returns a KeySetIterator over the provided keys.Range of keys.
func (s KeySet[K]) Iterate(ctx sdk.Context, r Ranger[K]) KeySetIterator[K] {
	mi := (Map[K, setObject])(s).Iterate(ctx, r)
	return (KeySetIterator[K])(mi)
}

// Close closes the KeySetIterator.
// No other operation is valid.
func (s KeySetIterator[K]) Close() { (Iterator[K, setObject])(s).Close() }

// Next moves the iterator onto the next key.
func (s KeySetIterator[K]) Next() { (Iterator[K, setObject])(s).Next() }

// Valid checks if the iterator is still valid.
func (s KeySetIterator[K]) Valid() bool { return (Iterator[K, setObject])(s).Valid() }

// Key returns the current iterator key.
func (s KeySetIterator[K]) Key() K { return (Iterator[K, setObject])(s).Key() }

// Keys consumes the iterator fully and returns all the available keys.
// The KeySetIterator is closed after this operation.
func (s KeySetIterator[K]) Keys() []K { return (Iterator[K, setObject])(s).Keys() }

// setObject represents a noop object used for sets,
// it also implements the ValueEncoder interface for itself.
type setObject struct{}

func (s setObject) Encode(_ setObject) []byte { return []byte{} }

func (s setObject) Decode(b []byte) setObject {
	if !bytes.Equal(b, []byte{}) {
		panic(fmt.Sprintf("invalid bytes: %s", b))
	}
	return setObject{}
}

func (s setObject) Stringify(_ setObject) string {
	return "setObject{}"
}

func (s setObject) Name() string {
	return "setObject"
}
