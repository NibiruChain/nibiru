package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPoolShareDenom(t *testing.T) {
	require.Equal(t, "matrix/pool/123", GetPoolShareDenom(123))
}
