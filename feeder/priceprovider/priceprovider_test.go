package priceprovider

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExpiringPriceProvider(t *testing.T) {
	t.Run("instantiation panic", func(t *testing.T) {
		require.Panics(t, func() {
			NewExpiringPriceProvider(nil, -1)
		})
	})

	t.Run("expired price", func(t *testing.T) {

	})

	t.Run("not expired price", func(t *testing.T) {

	})
}
