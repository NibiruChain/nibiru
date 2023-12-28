package keeper

import (
	"fmt"
	"time"

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
	PbMsg oracletypes.MsgEditOracleParams
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

	mergedParams := newParams.MergeOracleParams(params)
	k.UpdateParams(ctx, mergedParams)
	return nil
}

// MergeOracleParams: Takes the given oracle params and merges them into the
// existing partial params, keeping any existing values that are not set in the
// partial.
func (partial PartialOracleParams) MergeOracleParams(
	oracleParams oracletypes.Params,
) oracletypes.Params {
	if partial.PbMsg.VotePeriod != nil {
		oracleParams.VotePeriod = partial.PbMsg.VotePeriod.Uint64()
	}

	if partial.PbMsg.VoteThreshold != nil {
		oracleParams.VoteThreshold = *partial.PbMsg.VoteThreshold
	}

	if partial.PbMsg.RewardBand != nil {
		oracleParams.RewardBand = *partial.PbMsg.RewardBand
	}

	if partial.PbMsg.Whitelist != nil {
		whitelist := make([]asset.Pair, len(partial.PbMsg.Whitelist))
		for i, pair := range partial.PbMsg.Whitelist {
			whitelist[i] = asset.MustNewPair(pair)
		}

		oracleParams.Whitelist = whitelist
	}

	if partial.PbMsg.SlashFraction != nil {
		oracleParams.SlashFraction = *partial.PbMsg.SlashFraction
	}

	if partial.PbMsg.SlashWindow != nil {
		oracleParams.SlashWindow = partial.PbMsg.SlashWindow.Uint64()
	}

	if partial.PbMsg.MinValidPerWindow != nil {
		oracleParams.MinValidPerWindow = *partial.PbMsg.MinValidPerWindow
	}

	if partial.PbMsg.TwapLookbackWindow != nil {
		oracleParams.TwapLookbackWindow = time.Duration(partial.PbMsg.TwapLookbackWindow.Int64())
	}

	if partial.PbMsg.MinVoters != nil {
		oracleParams.MinVoters = partial.PbMsg.MinVoters.Uint64()
	}

	if partial.PbMsg.ValidatorFeeRatio != nil {
		oracleParams.ValidatorFeeRatio = *partial.PbMsg.ValidatorFeeRatio
	}

	return oracleParams
}
