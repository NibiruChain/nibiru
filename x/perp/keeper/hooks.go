package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
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
		indexTWAPPrice, err := k.PricefeedKeeper.GetCurrentTWAPPrice(ctx, pairMetadata.Pair.Token0, pairMetadata.Pair.Token1)
		if err != nil {
			ctx.Logger().Error("failed to fetch twap index price", "pairMetadata.Pair", pairMetadata.Pair, "error", err)
			continue
		}
		if indexTWAPPrice.Price.IsZero() {
			ctx.Logger().Error("index price is zero", "pairMetadata.Pair", pairMetadata.Pair)
			continue
		}
		markTWAPPrice, err := k.VpoolKeeper.GetCurrentTWAPPrice(ctx, pairMetadata.Pair)
		if err != nil {
			ctx.Logger().Error("failed to fetch twap mark price", "pairMetadata.Pair", pairMetadata.Pair, "error", err)
			continue
		}
		if markTWAPPrice.Price.IsZero() {
			ctx.Logger().Error("mark price is zero", "pairMetadata.Pair", pairMetadata.Pair)
			continue
		}
		epochInfo := k.EpochKeeper.GetEpochInfo(ctx, epochIdentifier)
		intervalsPerDay := (24 * time.Hour) / epochInfo.Duration
		fundingRate := markTWAPPrice.Price.Sub(indexTWAPPrice.Price).QuoInt64(int64(intervalsPerDay))

		if len(pairMetadata.CumulativePremiumFractions) > 0 {
			fundingRate = pairMetadata.CumulativePremiumFractions[len(pairMetadata.CumulativePremiumFractions)-1].Add(fundingRate)
		}
		pairMetadata.CumulativePremiumFractions = append(pairMetadata.CumulativePremiumFractions, fundingRate)
		k.PairMetadataState(ctx).Set(pairMetadata)
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
