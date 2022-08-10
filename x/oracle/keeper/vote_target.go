package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

// IsVoteTarget returns existence of a pair in the voting target list
func (k Keeper) IsVoteTarget(ctx sdk.Context, pair string) bool {
	return k.PairExists(ctx, pair)
}

// GetVoteTargets returns the voting target list on current vote period
func (k Keeper) GetVoteTargets(ctx sdk.Context) (voteTargets []string) {
	k.IteratePairs(ctx, func(pair string) bool {
		voteTargets = append(voteTargets, pair)
		return false
	})

	return voteTargets
}
