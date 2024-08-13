package keeper

import (
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// UpdateParams updates the oracle parameters
func (k Keeper) UpdateParams(ctx sdk.Context, params types.Params) {
	k.Params.Set(ctx, params)
}

// VotePeriod returns the number of blocks during which voting takes place.
func (k Keeper) VotePeriod(ctx sdk.Context) (res uint64) {
	params, _ := k.Params.Get(ctx)
	return params.VotePeriod
}

// VoteThreshold returns the minimum percentage of votes that must be received for a votes to pass.
func (k Keeper) VoteThreshold(ctx sdk.Context) (res sdk.Dec) {
	params, _ := k.Params.Get(ctx)
	return params.VoteThreshold
}

// MinVoters returns the minimum percentage of votes that must be received for a votes to pass.
func (k Keeper) MinVoters(ctx sdk.Context) (res uint64) {
	params, _ := k.Params.Get(ctx)
	return params.MinVoters
}

// RewardBand returns a maxium divergence that a price vote can have from the
// weighted median in the votes. If a vote lies within the valid range
// defined by:
//
//	μ := weightedMedian,
//	validRange := μ ± (μ * rewardBand / 2),
//
// then rewards are added to the validator performance.
// Note that if the reward band is smaller than 1 standard
// deviation, the band is taken to be 1 standard deviation.
func (k Keeper) RewardBand(ctx sdk.Context) (res sdk.Dec) {
	params, _ := k.Params.Get(ctx)
	return params.RewardBand
}

// Whitelist returns the pair list that can be activated
func (k Keeper) Whitelist(ctx sdk.Context) (res []asset.Pair) {
	params, _ := k.Params.Get(ctx)
	return params.Whitelist
}

// SlashFraction returns oracle voting penalty rate
func (k Keeper) SlashFraction(ctx sdk.Context) (res sdk.Dec) {
	params, _ := k.Params.Get(ctx)
	return params.SlashFraction
}

// SlashWindow returns the number of voting periods that specify a "slash window".
// After each slash window, all oracles that have missed more than the penalty
// threshold are slashed. Missing the penalty threshold is synonymous with
// submitting fewer valid votes than `MinValidPerWindow`.
func (k Keeper) SlashWindow(ctx sdk.Context) (res uint64) {
	params, _ := k.Params.Get(ctx)
	return params.SlashWindow
}

// MinValidPerWindow returns oracle slashing threshold
func (k Keeper) MinValidPerWindow(ctx sdk.Context) (res sdk.Dec) {
	params, _ := k.Params.Get(ctx)
	return params.MinValidPerWindow
}
