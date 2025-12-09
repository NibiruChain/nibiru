package collections

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Map represents a generic key-value storage with custom encoding for keys and
// values. It uses a specific namespace to avoid collisions and provides methods
// for basic CRUD operations.
type Map[K, V any] struct {
	kc KeyEncoder[K]
	vc ValueEncoder[V]

	prefix []byte
	sk     types.StoreKey

	typeName string
}

// NewMap creates a new Map instance with specified storage key, namespace, key
// encoder, and value encoder. It initializes a namespace-specific prefix and
// type name for value type V.
func NewMap[K, V any](
	sk types.StoreKey, namespace Namespace, kc KeyEncoder[K], vc ValueEncoder[V],
) Map[K, V] {
	return Map[K, V]{
		kc:     kc,
		vc:     vc,
		prefix: namespace.Prefix(),
		sk:     sk,
		//nolint
		typeName: vc.(ValueEncoder[V]).Name(), // go1.19 compiler bug
	}
}

func (m Map[K, V]) Insert(ctx sdk.Context, k K, v V) {
	m.GetStore(ctx).
		Set(m.kc.Encode(k), m.vc.Encode(v))
}

func (m Map[K, V]) Get(ctx sdk.Context, k K) (v V, err error) {
	vBytes := m.GetStore(ctx).Get(m.kc.Encode(k))
	if vBytes == nil {
		return v, fmt.Errorf("%w: '%s' with key %s", ErrNotFound, m.typeName, m.kc.Stringify(k))
	}

	return m.vc.Decode(vBytes), nil
}

func (m Map[K, V]) GetOr(ctx sdk.Context, key K, def V) (v V) {
	v, err := m.Get(ctx, key)
	if err == nil {
		return
	}

	return def
}

// Delete removes the key-value pair associated with the key from the map.
// Returns an error if the key does not exist.
func (m Map[K, V]) Delete(ctx sdk.Context, k K) error {
	kBytes := m.kc.Encode(k)
	store := m.GetStore(ctx)
	if !store.Has(kBytes) {
		return fmt.Errorf("%w: '%s' with key %s", ErrNotFound, m.typeName, m.kc.Stringify(k))
	}
	store.Delete(kBytes)

	return nil
}

// Iterate returns an iterator that traverses the elements of the map within the
// specified range. It utilizes the custom key encoder for navigating the
// underlying storage.
func (m Map[K, V]) Iterate(ctx sdk.Context, rng Ranger[K]) Iterator[K, V] {
	return iteratorFromRange[K, V](m.GetStore(ctx), rng, m.kc, m.vc)
}

// GetStore returns a namespaced version of the underlying KVStore for the map.
// It is used to access the store using the prefixed namespace.
func (m Map[K, V]) GetStore(ctx sdk.Context) sdk.KVStore {
	kvStore := ctx.KVStore(m.sk) // persistent store
	return prefix.NewStore(kvStore, m.prefix)
}

// MapTransient: A composed, or embedded, version of the `collections.Map` that
// maps to a transient KV store instead of a persistent one. A Transient KV Store
// is used for data that does not need to persist beyond the execution of the
// current block or transaction.
//
// This can include temporary calculations, intermediate state data in
// transactions or ephemeral data used in block processing. Data is a transient
// store is cleared after the block is processed.
//
// Transient KV stores have markedly lower costs for all operations (10% of the
// persistent cost) and a read cost per byte of zero.
type MapTransient[K, V any] struct {
	Map[K, V]
}

// GetStore returns a namespaced version of the underlying KVStore for the map.
// It is used to access the store using the prefixed namespace.
func (m MapTransient[K, V]) GetStore(ctx sdk.Context) sdk.KVStore {
	kvStore := ctx.TransientStore(m.sk)
	return prefix.NewStore(kvStore, m.prefix)
}

func NewMapTransient[K, V any](
	sk types.StoreKey, namespace Namespace, kc KeyEncoder[K], vc ValueEncoder[V],
) MapTransient[K, V] {
	return MapTransient[K, V]{
		Map: Map[K, V]{
			kc:     kc,
			vc:     vc,
			prefix: namespace.Prefix(),
			sk:     sk,
			//nolint
			typeName: vc.(ValueEncoder[V]).Name(), // go1.19 compiler bug
		},
	}
}
