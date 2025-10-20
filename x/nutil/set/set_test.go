package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var elementSlice = []string{"fire", "earth", "water", "air"}

func TestAdd(t *testing.T) {
	elements := New(elementSlice...)

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
	elements := New(elementSlice...)
	elem := "water"
	assert.True(t, elements.Has(elem))

	elements.Remove(elem)
	assert.False(t, elements.Has(elem))
}

func TestHas(t *testing.T) {
	elements := New(elementSlice...)

	assert.True(t, elements.Has("fire"))
	assert.True(t, elements.Has("water"))
	assert.True(t, elements.Has("air"))
	assert.True(t, elements.Has("earth"))
	assert.False(t, elements.Has(""))
	assert.False(t, elements.Has("foo"))
	assert.False(t, elements.Has("bar"))
}

func TestLen(t *testing.T) {
	elements := New(elementSlice...)
	assert.Equal(t, elements.Len(), 4)

	elements.Remove("fire")
	elements.Remove("water")
	assert.Equal(t, elements.Len(), 2)
}

func TestEquals(t *testing.T) {
	t.Run("equal: same elements different order", func(t *testing.T) {
		a := New("fire", "earth", "water", "air")
		b := New("air", "water", "earth", "fire")
		assert.True(t, a.Equals(b))
		assert.True(t, b.Equals(a)) // symmetry
		assert.True(t, a.Equals(a)) // reflexive
	})

	t.Run("not equal: different sizes", func(t *testing.T) {
		a := New("fire", "earth", "water", "air")
		b := New("fire", "earth", "water")
		assert.False(t, a.Equals(b))
		assert.False(t, b.Equals(a))
	})

	t.Run("not equal: same size but different members", func(t *testing.T) {
		a := New("fire", "earth", "water", "air")
		b := New("fire", "earth", "water", "lava")
		assert.False(t, a.Equals(b))
		assert.False(t, b.Equals(a))
	})

	t.Run("equal: empty vs empty", func(t *testing.T) {
		a := New[string]()          // empty map
		var b Set[string]           // nil map
		assert.True(t, a.Equals(b)) // len=0 on both; iteration over nil map is safe
		assert.True(t, b.Equals(a))
		assert.True(t, b.Equals(b))
	})

	t.Run("equal after mutations", func(t *testing.T) {
		a := New("fire")
		b := New[string]()
		b.Add("earth")
		b.Remove("earth")
		b.Add("fire")
		assert.True(t, a.Equals(b))
		assert.True(t, b.Equals(a))
	})

	t.Run("not equal after opposing mutations", func(t *testing.T) {
		a := New("fire", "earth")
		b := New("fire", "earth")
		a.Remove("earth")
		b.Add("water")
		assert.False(t, a.Equals(b))
		assert.False(t, b.Equals(a))
	})
}
