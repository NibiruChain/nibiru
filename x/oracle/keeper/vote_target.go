package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common"
)

// IsWhitelistedPair returns existence of a pair in the voting target list
func (k Keeper) IsWhitelistedPair(ctx sdk.Context, pair common.AssetPair) bool {
	return k.WhitelistedPairs.Has(ctx, pair)
}

// GetWhitelistedPairs returns the whitelisted pairs list on current vote period
func (k Keeper) GetWhitelistedPairs(ctx sdk.Context) []common.AssetPair {
	return k.WhitelistedPairs.Iterate(ctx, collections.Range[common.AssetPair]{}).Keys()
}
