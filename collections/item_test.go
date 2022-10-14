package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemEmpty(t *testing.T) {
	sk, ctx, _ := deps()
	item := NewItem[string](sk, 0, stringValue{})

	val, err := item.Get(ctx)
	assert.EqualValues(t, "", val)
	assert.Error(t, err)
}

func TestItemGetOr(t *testing.T) {
	sk, ctx, _ := deps()
	item := NewItem[string](sk, 0, stringValue{})

	val := item.GetOr(ctx, "default")
	assert.EqualValues(t, "default", val)
}

func TestItemSetAndGet(t *testing.T) {
	sk, ctx, _ := deps()
	item := NewItem[string](sk, 0, stringValue{})
	item.Set(ctx, "bar")
	val, err := item.Get(ctx)
	require.Nil(t, err)
	require.EqualValues(t, "bar", val)
}
