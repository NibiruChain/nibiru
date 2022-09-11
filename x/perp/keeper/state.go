package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// getLatestCumulativePremiumFraction returns the last cumulative premium fraction recorded for the
// specific pair.
func (k Keeper) getLatestCumulativePremiumFraction(
	ctx sdk.Context, pair common.AssetPair,
) (sdk.Dec, error) {
	pairMetadata, err := k.PairMetadata.Get(ctx, pair)
	if err != nil {
		k.Logger(ctx).Error(
			err.Error(),
			"pair",
			pair.String(),
		)
		return sdk.Dec{}, types.ErrPairMetadataNotFound
	}
	// this should never fail
	return pairMetadata.CumulativePremiumFractions[len(pairMetadata.CumulativePremiumFractions)-1], nil
}
