package evm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainConfigValidate(t *testing.T) {
	err := Validate()
	require.NoError(t, err)
}
