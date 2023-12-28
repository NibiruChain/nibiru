package keeper

import (
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

// Extends the Keeper with admin functions. Admin is syntactic sugar to separate
// admin calls off from the other Keeper methods.
//
// These Admin functions should:
// 1. Not be called in other methods in the x/perp module.
// 2. Only be callable from the x/sudo root or sudo contracts.
//
// The intention behind "admin" is to make it more obvious to the developer that
// an unsafe function is being used when it's called from "OracleKeeper.Admin"
type admin struct{ *Keeper }

type PartialOracleParams struct {
	VotePeriod         *sdkmath.Int `json:"vote_period,omitempty"`
	VoteThreshold      *sdk.Dec     `json:"vote_threshold,omitempty"`
	RewardBand         *sdk.Dec     `json:"reward_band,omitempty"`
	Whitelist          []string     `json:"whitelist,omitempty"`
	SlashFraction      *sdk.Dec     `json:"slash_fraction,omitempty"`
	SlashWindow        *sdkmath.Int `json:"slash_window,omitempty"`
	MinValidPerWindow  *sdk.Dec     `json:"min_valid_per_window,omitempty"`
	TwapLookbackWindow *sdkmath.Int `json:"twap_lookback_window,omitempty"`
	MinVoters          *sdkmath.Int `json:"min_voters,omitempty"`
	ValidatorFeeRatio  *sdk.Dec     `json:"validator_fee_ratio,omitempty"`
}

func (k admin) EditOracleParams(
	ctx sdk.Context, newParams PartialOracleParams, sender sdk.AccAddress,
) error {
	if err := k.SudoKeeper.CheckPermissions(sender, ctx); err != nil {
		return err
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		// TODO: use typed error
		return fmt.Errorf("get oracle params error: %s", err.Error())
	}

	mergedParams := newParams.mergeOracleParams(params)
	k.UpdateParams(ctx, mergedParams)
	return nil
}

// mergeOracleParams: Takes the givne oracle params and merges them into the
// existing partial params, keeping any existing values that are note set in the
// partial msg
func (msg PartialOracleParams) mergeOracleParams(
	oracleParams oracletypes.Params,
) oracletypes.Params {
	if msg.VotePeriod != nil {
		oracleParams.VotePeriod = msg.VotePeriod.Uint64()
	}

	if msg.VoteThreshold != nil {
		oracleParams.VoteThreshold = *msg.VoteThreshold
	}

	if msg.RewardBand != nil {
		oracleParams.RewardBand = *msg.RewardBand
	}

	if msg.Whitelist != nil {
		whitelist := make([]asset.Pair, len(msg.Whitelist))
		for i, pair := range msg.Whitelist {
			whitelist[i] = asset.MustNewPair(pair)
		}

		oracleParams.Whitelist = whitelist
	}

	if msg.SlashFraction != nil {
		oracleParams.SlashFraction = *msg.SlashFraction
	}

	if msg.SlashWindow != nil {
		oracleParams.SlashWindow = msg.SlashWindow.Uint64()
	}

	if msg.MinValidPerWindow != nil {
		oracleParams.MinValidPerWindow = *msg.MinValidPerWindow
	}

	if msg.TwapLookbackWindow != nil {
		oracleParams.TwapLookbackWindow = time.Duration(msg.TwapLookbackWindow.Int64())
	}

	if msg.MinVoters != nil {
		oracleParams.MinVoters = msg.MinVoters.Uint64()
	}

	if msg.ValidatorFeeRatio != nil {
		oracleParams.ValidatorFeeRatio = *msg.ValidatorFeeRatio
	}

	return oracleParams
}
