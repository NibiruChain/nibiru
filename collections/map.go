package collections

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Map[K, V any] struct {
	kc KeyEncoder[K]
	vc ValueEncoder[V]

	prefix []byte
	sk     sdk.StoreKey

	typeName string
}

func NewMap[K, V any](sk sdk.StoreKey, namespace Namespace, kc KeyEncoder[K], vc ValueEncoder[V]) Map[K, V] {
	return Map[K, V]{
		kc:     kc,
		vc:     vc,
		prefix: namespace.Prefix(),
		sk:     sk,
	}
}

func (m Map[K, V]) Insert(ctx sdk.Context, k K, v V) {
	m.getStore(ctx).
		Set(m.kc.Encode(k), m.vc.Encode(v))
}

func (m Map[K, V]) Get(ctx sdk.Context, k K) (v V, err error) {
	vBytes := m.getStore(ctx).Get(m.kc.Encode(k))
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

func (m Map[K, V]) Delete(ctx sdk.Context, k K) error {
	kBytes := m.kc.Encode(k)
	store := m.getStore(ctx)
	if !store.Has(kBytes) {
		return fmt.Errorf("%w: '%s' with key %s", ErrNotFound, m.typeName, m.kc.Stringify(k))
	}
	store.Delete(kBytes)

	return nil
}

func (m Map[K, V]) Iterate(ctx sdk.Context, rng Ranger[K]) Iterator[K, V] {
	return iteratorFromRange[K, V](m.getStore(ctx), rng, m.kc, m.vc)
}

func (m Map[K, V]) getStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(m.sk), m.prefix)
}
