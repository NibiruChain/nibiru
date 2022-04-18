package keeper

import (
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

var _ types.QueryServer = Keeper{}
