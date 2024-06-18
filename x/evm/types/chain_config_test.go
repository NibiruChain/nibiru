package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/evm/types"
)

func TestChainConfigValidate(t *testing.T) {
	err := types.Validate()
	require.NoError(t, err)
}
