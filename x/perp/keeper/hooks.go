package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func (k Keeper) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
}

func (k Keeper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, _ int64) {
	params := k.GetParams(ctx)
	if epochIdentifier != params.EpochIdentifier || params.Stopped {
		return
	}

	for _, pairMetadata := range k.PairMetadataState(ctx).GetAll() {
		if !k.VpoolKeeper.ExistsPool(ctx, pairMetadata.Pair) {
			ctx.Logger().Error("no pool for pair found", "pairMetadata.Pair", pairMetadata.Pair)
			continue
		}

		indexTWAP, err := k.PricefeedKeeper.GetCurrentTWAP(ctx, pairMetadata.Pair.Token0, pairMetadata.Pair.Token1)
		if err != nil {
			ctx.Logger().Error("failed to fetch twap index price", "pairMetadata.Pair", pairMetadata.Pair, "error", err)
			continue
		}
		if indexTWAP.Price.IsZero() {
			ctx.Logger().Error("index price is zero", "pairMetadata.Pair", pairMetadata.Pair)
			continue
		}

		markTWAP, err := k.VpoolKeeper.GetCurrentTWAP(ctx, pairMetadata.Pair)
		if err != nil {
			ctx.Logger().Error("failed to fetch twap mark price", "pairMetadata.Pair", pairMetadata.Pair, "error", err)
			continue
		}
		if markTWAP.Price.IsZero() {
			ctx.Logger().Error("mark price is zero", "pairMetadata.Pair", pairMetadata.Pair)
			continue
		}

		epochInfo := k.EpochKeeper.GetEpochInfo(ctx, epochIdentifier)
		intervalsPerDay := (24 * time.Hour) / epochInfo.Duration
		fundingRate := markTWAP.Price.Sub(indexTWAP.Price).QuoInt64(int64(intervalsPerDay))

		// If there is a previous cumulative funding rate, add onto that one. Otherwise, the funding rate is the first cumulative funding rate.
		cumulativeFundingRate := fundingRate
		if len(pairMetadata.CumulativePremiumFractions) > 0 {
			cumulativeFundingRate = pairMetadata.CumulativePremiumFractions[len(pairMetadata.CumulativePremiumFractions)-1].Add(fundingRate)
		}

		pairMetadata.CumulativePremiumFractions = append(pairMetadata.CumulativePremiumFractions, cumulativeFundingRate)
		k.PairMetadataState(ctx).Set(pairMetadata)

		if err = ctx.EventManager().EmitTypedEvent(&types.FundingRateChangedEvent{
			Pair:                  pairMetadata.Pair.String(),
			MarkPrice:             markTWAP.Price,
			IndexPrice:            indexTWAP.Price,
			LatestFundingRate:     fundingRate,
			CumulativeFundingRate: cumulativeFundingRate,
			BlockHeight:           ctx.BlockHeight(),
			BlockTimeMs:           ctx.BlockTime().UnixMilli(),
		}); err != nil {
			ctx.Logger().Error("failed to emit FundingRateChangedEvent", "pairMetadata.Pair", pairMetadata.Pair, "error", err)
			continue
		}
	}
}

// ___________________________________________________________________________________________________

// Hooks wrapper struct for perps keeper.
type Hooks struct {
	k Keeper
}

var _ epochstypes.EpochHooks = Hooks{}

// Return the wrapper struct.
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// epochs hooks.
func (h Hooks) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	h.k.BeforeEpochStart(ctx, epochIdentifier, epochNumber)
}

func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	h.k.AfterEpochEnd(ctx, epochIdentifier, epochNumber)
}
