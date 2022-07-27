package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
)

type PricefeedKeeper interface {
	GetCurrentPrice(ctx sdk.Context, token0 string, token1 string) (
		pftypes.CurrentPrice, error,
	)
	IsActivePair(ctx sdk.Context, pairID string) bool
}
