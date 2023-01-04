package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/x/common"
)

var elementSlice = []string{"fire", "earth", "water", "air"}

func TestAdd(t *testing.T) {
	elements := common.NewStringSet(elementSlice...)
	assert.False(t, elements.Has("lava"))
	assert.False(t, elements.Has("mud"))
	elements.Add("lava")
	elements.Add("mud")
	assert.True(t, elements.Has("lava"))
	assert.True(t, elements.Has("mud"))
	assert.Equal(t, 6, elements.Len())

	// Add blank string
	elements.Add("")
	assert.True(t, elements.Has(""))
	assert.Equal(t, 7, elements.Len())
}

func TestRemove(t *testing.T) {
	elements := common.NewStringSet(elementSlice...)
	elem := "water"
	assert.True(t, elements.Has(elem))
	elements.Remove(elem)
	assert.False(t, elements.Has(elem))
}

func TestHas(t *testing.T) {
	elements := common.NewStringSet(elementSlice...)
	assert.True(t, elements.Has("fire"))
	assert.True(t, elements.Has("water"))
	assert.True(t, elements.Has("air"))
	assert.True(t, elements.Has("earth"))
	assert.False(t, elements.Has(""))
	assert.False(t, elements.Has("foo"))
	assert.False(t, elements.Has("bar"))
}

func TestLen(t *testing.T) {
	elements := common.NewStringSet(elementSlice...)
	assert.Equal(t, elements.Len(), 4)

	elements.Remove("fire")
	elements.Remove("water")
	assert.Equal(t, elements.Len(), 2)
}
