package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/collections"

	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

// IsWhitelistedPair returns existence of a pair in the voting target list
func (k Keeper) IsWhitelistedPair(ctx sdk.Context, pair types.Pair) bool {
	return k.WhitelistedPairs.Has(ctx, pair)
}

// GetWhitelistedPairs returns the whitelisted pairs list on current vote period
func (k Keeper) GetWhitelistedPairs(ctx sdk.Context) []types.Pair {
	return k.WhitelistedPairs.Iterate(ctx, collections.Range[types.Pair]{}).Keys()
}
