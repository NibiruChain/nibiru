package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPoolShareBaseDenom(t *testing.T) {
	require.Equal(t, "matrix/pool/123", GetPoolShareBaseDenom(123))
}

func TestGetPoolShareDisplayDenom(t *testing.T) {
	require.Equal(t, "MATRIX-POOL-123", GetPoolShareDisplayDenom(123))
}
