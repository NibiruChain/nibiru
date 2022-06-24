package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
)

func (k Keeper) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
}

func (k Keeper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, _ int64) {
	params := k.GetParams(ctx)
	if epochIdentifier != params.EpochIdentifier || params.Stopped {
		return
	}
	for _, md := range k.PairMetadataState(ctx).GetAll() {
		assetPair, err := common.NewAssetPairFromStr(md.Pair)
		if err != nil {
			ctx.Logger().Error("invalid asset pair", "assetPair", md.Pair, "error", err)
			continue
		}
		if !k.VpoolKeeper.ExistsPool(ctx, assetPair) {
			ctx.Logger().Error("no pool for pair found", "assetPair", assetPair)
			continue
		}
		indexTWAPPrice, err := k.PricefeedKeeper.GetCurrentTWAPPrice(ctx, assetPair.Token0, assetPair.Token1)
		if err != nil {
			ctx.Logger().Error("failed to fetch twap index price", "assetPair", assetPair, "error", err)
			continue
		}
		if indexTWAPPrice.Price.IsZero() {
			ctx.Logger().Error("index price is zero", "assetPair", assetPair)
			continue
		}
		markTWAPPrice, err := k.VpoolKeeper.GetCurrentTWAPPrice(ctx, assetPair.Token0, assetPair.Token1)
		if err != nil {
			ctx.Logger().Error("failed to fetch twap mark price", "assetPair", assetPair, "error", err)
			continue
		}
		if markTWAPPrice.Price.IsZero() {
			ctx.Logger().Error("mark price is zero", "assetPair", assetPair)
			continue
		}
		epochInfo := k.EpochKeeper.GetEpochInfo(ctx, epochIdentifier)
		intervalsPerDay := (24 * time.Hour) / epochInfo.Duration
		fundingRate := markTWAPPrice.Price.Sub(indexTWAPPrice.Price).QuoInt64(int64(intervalsPerDay))

		if len(md.CumulativePremiumFractions) > 0 {
			fundingRate = md.CumulativePremiumFractions[len(md.CumulativePremiumFractions)-1].Add(fundingRate)
		}
		md.CumulativePremiumFractions = append(md.CumulativePremiumFractions, fundingRate)
		k.PairMetadataState(ctx).Set(md)
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
