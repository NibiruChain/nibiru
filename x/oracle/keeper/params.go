package keeper

import (
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/oracle/types"

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

// mergeOracleParams takes the oracle params from the wasm msg and merges them into the existing params
// keeping any existing values if not set in the wasm msg
func mergeOracleParams(msg *types.MsgEditOracleParams, oracleParams types.Params) types.Params {
	if msg.Params.VotePeriod != 0 {
		oracleParams.VotePeriod = msg.Params.VotePeriod
	}

	if msg.Params.VoteThreshold != nil && !msg.Params.VoteThreshold.IsNil() {
		oracleParams.VoteThreshold = *msg.Params.VoteThreshold
	}

	if msg.Params.RewardBand != nil && !msg.Params.RewardBand.IsNil() {
		oracleParams.RewardBand = *msg.Params.RewardBand
	}

	if msg.Params.Whitelist != nil && len(msg.Params.Whitelist) != 0 {
		oracleParams.Whitelist = msg.Params.Whitelist
	}

	if msg.Params.SlashFraction != nil && !msg.Params.SlashFraction.IsNil() {
		oracleParams.SlashFraction = *msg.Params.SlashFraction
	}

	if msg.Params.SlashWindow != 0 {
		oracleParams.SlashWindow = msg.Params.SlashWindow
	}

	if msg.Params.MinValidPerWindow != nil && !msg.Params.MinValidPerWindow.IsNil() {
		oracleParams.MinValidPerWindow = *msg.Params.MinValidPerWindow
	}

	if msg.Params.TwapLookbackWindow != nil {
		oracleParams.TwapLookbackWindow = *msg.Params.TwapLookbackWindow
	}

	if msg.Params.MinVoters != 0 {
		oracleParams.MinVoters = msg.Params.MinVoters
	}

	if msg.Params.ValidatorFeeRatio != nil && !msg.Params.ValidatorFeeRatio.IsNil() {
		oracleParams.ValidatorFeeRatio = *msg.Params.ValidatorFeeRatio
	}

	if msg.Params.ExpirationBlocks != 0 {
		oracleParams.ExpirationBlocks = msg.Params.ExpirationBlocks
	}

	return oracleParams
}
