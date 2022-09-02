package websocket

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_genericJSONHandler(t *testing.T) {
	type x struct {
		H string `json:"h"`
	}
	expected := x{H: "1"}

	f := genericJSONHandler(func(got x) {
		require.Equal(t, expected, got)
	})

	// all works fine, and we assert values are equal
	require.NotPanics(t, func() {
		f([]byte(`{"h": "1"}`))
	})

	// assert that unexpected end of json input does not panic
	require.NotPanics(t, func() {
		f([]byte(`{"h": "1"`))
	})

	// assert that any other json error fails
	require.Panics(t, func() {
		f([]byte(`{"h": 1}`)) // type error
	})
}
