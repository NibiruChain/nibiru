package types

import (
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PricefeedKeeper interface {
	GetCurrentPrice(ctx sdk.Context, token0 string, token1 string) (
		pftypes.CurrentPrice, error,
	)
}
