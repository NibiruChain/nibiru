package keeper

import (
	"github.com/NibiruChain/nibiru/collections"
	"github.com/NibiruChain/nibiru/collections/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// NOTE(mercilex): just to showcase easiness of implementing complex queries
func (k Keeper) AllPairPositions(ctx sdk.Context, symbol common.AssetPair) []collections.KeyValue[keys.Pair[common.AssetPair, keys.StringKey], types.Position, *types.Position] {
	prefix := keys.PairPrefix[common.AssetPair, keys.StringKey](symbol)
	return k.Positions.Iterate(
		ctx,
		keys.NewRange[keys.Pair[common.AssetPair, keys.StringKey]]().
			Prefix(prefix),
	).KeyValues()

}

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
