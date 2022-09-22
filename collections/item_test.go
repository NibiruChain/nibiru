package collections

import (
	"testing"

	wellknown "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemEmpty(t *testing.T) {
	sk, ctx, cdc := deps()
	item := NewItem[wellknown.BytesValue](cdc, sk, 0)

	val, err := item.Get(ctx)
	assert.EqualValues(t, wellknown.BytesValue{}, val)
	assert.Error(t, err)
}

func TestItemGetWithDefault(t *testing.T) {
	sk, ctx, cdc := deps()
	item := NewItem[wellknown.BytesValue](cdc, sk, 0)

	val := item.GetOr(ctx, wellknown.BytesValue{Value: []byte("foo")})
	assert.EqualValues(t, wellknown.BytesValue{Value: []byte("foo")}, val)
}

func TestItemSetAndGet(t *testing.T) {
	sk, ctx, cdc := deps()
	item := NewItem[wellknown.BytesValue](cdc, sk, 0)
	item.Set(ctx, wellknown.BytesValue{Value: []byte("bar")})
	val, err := item.Get(ctx)
	require.Nil(t, err)
	require.EqualValues(t, wellknown.BytesValue{Value: []byte("bar")}, val)
}
