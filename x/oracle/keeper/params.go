package keeper

import (
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VotePeriod returns the number of blocks during which voting takes place.
func (k Keeper) VotePeriod(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyVotePeriod, &res)
	return
}

// VoteThreshold returns the minimum percentage of votes that must be received for a ballot to pass.
func (k Keeper) VoteThreshold(ctx sdk.Context) (res sdk.Dec) {
	k.paramSpace.Get(ctx, types.KeyVoteThreshold, &res)
	return
}

// RewardBand returns a maxium divergence that a price vote can have from the
// weighted median in the ballot. If a vote lies within the valid range
// defined by:
//
//	μ := weightedMedian,
//	validRange := μ ± (μ * rewardBand / 2),
//
// then rewards are added to the validator performance.
// Note that if the reward band is smaller than 1 standard
// deviation, the band is taken to be 1 standard deviation.a price
func (k Keeper) RewardBand(ctx sdk.Context) (res sdk.Dec) {
	k.paramSpace.Get(ctx, types.KeyRewardBand, &res)
	return
}

// Whitelist returns the pair list that can be activated
func (k Keeper) Whitelist(ctx sdk.Context) (res []asset.Pair) {
	k.paramSpace.Get(ctx, types.KeyWhitelist, &res)
	return
}

// SetWhitelist store new whitelist to param store
// this function is only for test purpose
func (k Keeper) SetWhitelist(ctx sdk.Context, whitelist []asset.Pair) {
	k.paramSpace.Set(ctx, types.KeyWhitelist, whitelist)
}

// SlashFraction returns oracle voting penalty rate
func (k Keeper) SlashFraction(ctx sdk.Context) (res sdk.Dec) {
	k.paramSpace.Get(ctx, types.KeySlashFraction, &res)
	return
}

// SlashWindow returns the number of voting periods that specify a "slash window".
// After each slash window, all oracles that have missed more than the penalty
// threshold are slashed. Missing the penalty threshold is synonymous with
// submitting fewer valid votes than `MinValidPerWindow`.
func (k Keeper) SlashWindow(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeySlashWindow, &res)
	return
}

// MinValidPerWindow returns oracle slashing threshold
func (k Keeper) MinValidPerWindow(ctx sdk.Context) (res sdk.Dec) {
	k.paramSpace.Get(ctx, types.KeyMinValidPerWindow, &res)
	return
}

// GetParams returns the total set of oracle parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the total set of oracle parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
