package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

// IsVoteTarget returns existence of a pair in the voting target list
func (k Keeper) IsVoteTarget(ctx sdk.Context, pair string) bool {
	_, err := k.GetTobinTax(ctx, pair)
	return err == nil
}

// GetVoteTargets returns the voting target list on current vote period
func (k Keeper) GetVoteTargets(ctx sdk.Context) (voteTargets []string) {
	k.IterateTobinTaxes(ctx, func(pair string, _ sdk.Dec) bool {
		voteTargets = append(voteTargets, pair)
		return false
	})

	return voteTargets
}
