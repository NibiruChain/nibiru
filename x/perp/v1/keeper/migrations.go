package keeper

import (
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/v1/amm/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v1/types"
)

func From2To3(perpKeeper Keeper, perpammKeeper types.PerpAmmKeeper) module.MigrationHandler {
	return func(ctx sdk.Context) error {
		k, ok := perpammKeeper.(perpammkeeper.Keeper)
		if !ok {
			panic("market keeper is not perpammkeeper.Keeper")
		}

		iterator := k.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
		for _, pool := range iterator {
			sumLong := sdk.ZeroDec()
			sumShort := sdk.ZeroDec()

			positions := perpKeeper.Positions.Iterate(ctx, collections.PairRange[asset.Pair, sdk.AccAddress]{}).Values()
			for _, position := range positions {
				if position.Size_.IsPositive() {
					sumLong = sumLong.Add(position.Size_)
				} else {
					sumShort = sumShort.Add(position.Size_.Neg())
				}
			}

			pool.TotalLong = sumLong
			pool.TotalShort = sumShort
			pool.PegMultiplier = sdk.OneDec()
			k.Pools.Insert(ctx, pool.Pair, pool)
		}

		return nil
	}
}
