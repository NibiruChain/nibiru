package keeper

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

// Sudo extends the Keeper with sudo functions. See sudo.go. Sudo is syntactic
// sugar to separate admin calls off from the other Keeper methods.
//
// These Sudo functions should:
// 1. Not be called in other methods in the x/perp module.
// 2. Only be callable by the x/sudo root or sudo contracts.
//
// The intention behind "Keeper.Sudo()" is to make it more obvious to the
// developer that an unsafe function is being used when it's called.
func (k Keeper) Sudo() sudoExtension { return sudoExtension{k} }

type sudoExtension struct{ Keeper }

// ------------------------------------------------------------------
// Admin.EditOracleParams

func (k sudoExtension) EditOracleParams(
	ctx sdk.Context, newParams oracletypes.MsgEditOracleParams,
	sender sdk.AccAddress,
) (paramsAfter oracletypes.Params, err error) {
	if err := k.sudoKeeper.CheckPermissions(sender, ctx); err != nil {
		return paramsAfter, err
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return paramsAfter, fmt.Errorf("%w: failed to read oracle params", err)
	}

	paramsAfter = MergeOracleParams(newParams, params)
	k.UpdateParams(ctx, paramsAfter)
	return paramsAfter, paramsAfter.Validate()
}

// MergeOracleParams: Takes the given oracle params and merges them into the
// existing partial params, keeping any existing values that are not set in the
// partial.
func MergeOracleParams(
	partial oracletypes.MsgEditOracleParams,
	oracleParams oracletypes.Params,
) oracletypes.Params {
	if partial.VotePeriod != nil {
		oracleParams.VotePeriod = partial.VotePeriod.Uint64()
	}

	if partial.VoteThreshold != nil {
		oracleParams.VoteThreshold = *partial.VoteThreshold
	}

	if partial.RewardBand != nil {
		oracleParams.RewardBand = *partial.RewardBand
	}

	if partial.Whitelist != nil {
		whitelist := make([]asset.Pair, len(partial.Whitelist))
		for i, pair := range partial.Whitelist {
			whitelist[i] = asset.MustNewPair(pair)
		}

		oracleParams.Whitelist = whitelist
	}

	if partial.SlashFraction != nil {
		oracleParams.SlashFraction = *partial.SlashFraction
	}

	if partial.SlashWindow != nil {
		oracleParams.SlashWindow = partial.SlashWindow.Uint64()
	}

	if partial.MinValidPerWindow != nil {
		oracleParams.MinValidPerWindow = *partial.MinValidPerWindow
	}

	if partial.TwapLookbackWindow != nil {
		oracleParams.TwapLookbackWindow = time.Duration(partial.TwapLookbackWindow.Int64())
	}

	if partial.MinVoters != nil {
		oracleParams.MinVoters = partial.MinVoters.Uint64()
	}

	if partial.ValidatorFeeRatio != nil {
		oracleParams.ValidatorFeeRatio = *partial.ValidatorFeeRatio
	}

	return oracleParams
}
