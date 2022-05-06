package v1

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName           = "perp"
	VaultModuleAccount   = "vault"
	PerpEFModuleAccount  = "perp_ef"
	FeePoolModuleAccount = "fee_pool"
)

func ZeroPosition(ctx sdk.Context, vpair string, trader string) *Position {
	return &Position{
		Address:                             trader,
		Pair:                                vpair,
		Size_:                               sdk.ZeroInt(),
		Margin:                              sdk.ZeroInt(),
		OpenNotional:                        sdk.ZeroInt(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroInt(),
		LiquidityHistoryIndex:               0,
		BlockNumber:                         ctx.BlockHeight(),
	}
}
