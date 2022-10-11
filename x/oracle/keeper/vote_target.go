package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections"
)

// IsVoteTarget returns existence of a pair in the voting target list
func (k Keeper) IsVoteTarget(ctx sdk.Context, pair string) bool {
	return k.Pairs.Has(ctx, pair)
}

// GetVoteTargets returns the voting target list on current vote period
func (k Keeper) GetVoteTargets(ctx sdk.Context) (voteTargets []string) {
	return k.Pairs.Iterate(ctx, collections.Range[string]{}).Keys()
}
