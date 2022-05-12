package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// TODO test: ClearPosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) ClearPosition(ctx sdk.Context, pair common.TokenPair, trader string) error {
	return k.Positions().Update(ctx, &types.Position{
		Address:                             trader,
		Pair:                                pair.String(),
		Size_:                               sdk.ZeroDec(),
		Margin:                              sdk.ZeroDec(),
		OpenNotional:                        sdk.ZeroDec(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
		LiquidityHistoryIndex:               0,
		BlockNumber:                         ctx.BlockHeight(),
	})
}

func (k Keeper) GetPosition(
	ctx sdk.Context, pair common.TokenPair, owner string,
) (*types.Position, error) {
	return k.Positions().Get(ctx, pair, owner)
}

func (k Keeper) SetPosition(
	ctx sdk.Context, pair common.TokenPair, owner string,
	position *types.Position) {
	k.Positions().Set(ctx, pair, owner, position)
}
