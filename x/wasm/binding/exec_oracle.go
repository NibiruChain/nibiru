package binding

import (
	"fmt"
	"github.com/NibiruChain/nibiru/x/common/asset"
	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ExecutorOracle struct {
	Oracle oraclekeeper.Keeper
}

func (o ExecutorOracle) SetOracleParams(msg *cw_struct.OracleParams, ctx sdk.Context) error {
	params, err := o.Oracle.Params.Get(ctx)
	if err != nil {
		return fmt.Errorf("get oracle params error: %s", err.Error())
	}

	mergedParams := mergeOracleParams(msg, params)

	o.Oracle.UpdateParams(ctx, mergedParams)
	return nil
}

// mergeOracleParams takes the oracle params from the wasm msg and merges them into the existing params
// keeping any existing values if not set in the wasm msg
func mergeOracleParams(msg *cw_struct.OracleParams, oracleParams oracletypes.Params) oracletypes.Params {
	if msg.VotePeriod != nil {
		oracleParams.VotePeriod = *msg.VotePeriod
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
		oracleParams.SlashWindow = *msg.SlashWindow
	}

	if msg.MinValidPerWindow != nil {
		oracleParams.MinValidPerWindow = *msg.MinValidPerWindow
	}

	if msg.TwapLookbackWindow != nil {
		oracleParams.TwapLookbackWindow = *msg.TwapLookbackWindow
	}

	if msg.MinVoters != nil {
		oracleParams.MinVoters = *msg.MinVoters
	}

	if msg.ValidatorFeeRatio != nil {
		oracleParams.ValidatorFeeRatio = *msg.ValidatorFeeRatio
	}

	return oracleParams
}
