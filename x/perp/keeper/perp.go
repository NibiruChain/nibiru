package keeper

import (
	types "github.com/NibiruChain/nibiru/x/perp/types/v1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ types.IClearingHouse = (*Keeper)(nil)
)

// TODO test: ClearPosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) ClearPosition(ctx sdk.Context, vpool types.IVirtualPool, trader string) error {
	return k.Positions().Update(ctx, &types.Position{
		Address:                             trader,
		Pair:                                vpool.Pair(),
		Size_:                               sdk.ZeroInt(),
		Margin:                              sdk.ZeroInt(),
		OpenNotional:                        sdk.ZeroInt(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroInt(),
		LiquidityHistoryIndex:               0,
		BlockNumber:                         ctx.BlockHeight(),
	})
}

func (k Keeper) GetPosition(
	ctx sdk.Context, vpool types.IVirtualPool, owner string,
) (*types.Position, error) {
	return k.Positions().Get(ctx, vpool.Pair(), owner)
}

func (k Keeper) SetPosition(
	ctx sdk.Context, vpool types.IVirtualPool, owner string,
	position *types.Position) {
	k.Positions().Set(ctx, vpool.Pair(), owner, position)
}
