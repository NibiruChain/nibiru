package cligen

import (
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"testing"
)

func TestCliGen(t *testing.T) {
	cligen := NewCliGen().
		ForMessage(&types.MsgPostPrice{}).WithParams()
}
