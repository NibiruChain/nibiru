package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
)

// IsWhitelistedPair returns existence of a pair in the voting target list
func (k Keeper) IsWhitelistedPair(ctx sdk.Context, pair string) bool {
	return k.WhitelistedPairs.Has(ctx, pair)
}

// GetWhitelistedPairs returns the voting target list on current vote period
func (k Keeper) GetWhitelistedPairs(ctx sdk.Context) (voteTargets []string) {
	return k.WhitelistedPairs.Iterate(ctx, collections.Range[string]{}).Keys()
}
