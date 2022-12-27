package cligen

import (
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"testing"
)

func TestCliGen(t *testing.T) {
	NewCliGen().
		ForMessage(&types.MsgPostPrice{}).
		WithParams(Params{
			{Name: "token0"},
			{Name: "token1"},
			{Name: "price"},
			{Name: "expiry"},
		}).Generate()
}
