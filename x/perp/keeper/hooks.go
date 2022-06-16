package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
)

func (k Keeper) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
}

func (k Keeper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	// FIXME: should this be moved to BeforeEpochStart??
	params := k.GetParams(ctx)
	if epochIdentifier != params.DistrEpochIdentifier {
		return
	}
	for _, md := range k.PairMetadata().GetAll(ctx) {
		assetPair, err := common.NewAssetPairFromStr(md.Pair)
		if err != nil {
			// FIXME: should we panic instead??
			continue
		}
		if !k.VpoolKeeper.ExistsPool(ctx , assetPair) {
			// FIXME: should we panic instead??
			continue
		}
		indexTWAPPrice, err := k.PricefeedKeeper.GetCurrentTWAPPrice(ctx, assetPair.Token0, assetPair.Token1)
		if err != nil {
			// FIXME: should we panic instead??
			continue
		}
		marketTWAPPrice, err := k.VpoolKeeper.GetCurrentTWAPPrice(ctx, assetPair.Token0, assetPair.Token1)
		fundingRate := marketTWAPPrice.Price.Sub(indexTWAPPrice.Price).Quo(sdk.NewDec(24))
		md.CumulativePremiumFractions = append(md.CumulativePremiumFractions, fundingRate)
		k.PairMetadata().Set(ctx, md)
	}
}



// ___________________________________________________________________________________________________

// Hooks wrapper struct for incentives keeper.
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
