package keeper

import (
	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common/asset"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func From1To2(k Keeper, perpKeeper perpkeeper.Keeper) module.MigrationHandler {
	return func(ctx sdk.Context) error {
		iterator := k.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
		for _, pool := range iterator {
			sumBias := sdk.ZeroDec()

			positions := perpKeeper.Positions.Iterate(ctx, collections.PairRange[asset.Pair, sdk.AccAddress]{}).Values()
			for _, position := range positions {
				sumBias = sumBias.Add(position.Size_)
			}

			pool.Bias = sumBias
			k.Pools.Insert(ctx, pool.Pair, pool)
		}

		return nil
	}
}
