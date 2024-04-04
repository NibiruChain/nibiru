package perp

import (
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// EndBlocker Called every block to store a snapshot of the perpamm.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	iter, err := k.AMMs.Iterate(ctx, &collections.Range[collections.Pair[asset.Pair, uint64]]{})
	if err != nil {
		k.Logger(ctx).Error("failed iterating amms", "error", err)
		return []abci.ValidatorUpdate{}
	}
	values, err := iter.Values()
	if err != nil {
		k.Logger(ctx).Error("failed getting amm values", "error", err)
		return []abci.ValidatorUpdate{}
	}

	for _, amm := range values {
		market, err := k.GetMarket(ctx, amm.Pair)
		if err != nil {
			k.Logger(ctx).Error("failed to fetch market", "pair", amm.Pair, "error", err)
			continue
		}

		// only snapshot enabled markets
		if !market.Enabled {
			continue
		}

		snapshot := types.ReserveSnapshot{
			Amm:         amm,
			TimestampMs: ctx.BlockTime().UnixMilli(),
		}
		k.ReserveSnapshots.Set(ctx, collections.Join(amm.Pair, ctx.BlockTime()), snapshot)

		markTwap, err := k.CalcTwap(ctx, amm.Pair, types.TwapCalcOption_SPOT, types.Direction_DIRECTION_UNSPECIFIED, sdkmath.LegacyZeroDec(), market.TwapLookbackWindow)
		if err != nil {
			k.Logger(ctx).Error("failed to fetch twap mark price", "market.Pair", market.Pair, "error", err)
			continue
		}

		if markTwap.IsNil() || markTwap.IsZero() {
			k.Logger(ctx).Error("mark price is zero", "market.Pair", market.Pair)
			continue
		}

		var indexTwap sdkmath.LegacyDec
		indexTwap, err = k.OracleKeeper.GetExchangeRateTwap(ctx, market.OraclePair)
		if err != nil {
			k.Logger(ctx).Error("failed to fetch twap index price", "market.Pair", market.Pair, "market.OraclePair", market.OraclePair, "error", err)
			indexTwap = sdkmath.LegacyOneDec().Neg()
		}

		if indexTwap.IsNil() {
			k.Logger(ctx).Error("index price is zero", "market.Pair", market.Pair, "market.OraclePair", market.OraclePair)
			continue
		}

		_ = ctx.EventManager().EmitTypedEvent(&types.AmmUpdatedEvent{
			FinalAmm:       amm,
			MarkPriceTwap:  markTwap,
			IndexPriceTwap: indexTwap,
		})

		_ = ctx.EventManager().EmitTypedEvent(&types.MarketUpdatedEvent{
			FinalMarket: market,
		})
	}

	return []abci.ValidatorUpdate{}
}
