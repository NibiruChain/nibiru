package cligen

import (
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCliGen(t *testing.T) {
	cmd := NewCliGen().
		ForMessage(&types.MsgPostPrice{}).
		WithParams(Params{
			{Name: "token0", Mandatory: true},
			{Name: "token1", Mandatory: true},
			{Name: "price"},
			{Name: "expiry"},
		}).Generate()

	require.Equal(t, "postprice [token0] [token1]", cmd.Use)
}
