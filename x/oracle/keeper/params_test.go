package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestParams(t *testing.T) {
	input := CreateTestFixture(t)

	// Test default params setting
	input.OracleKeeper.Params.Set(input.Ctx, types.DefaultParams())
	params, err := input.OracleKeeper.Params.Get(input.Ctx)
	require.NoError(t, err)
	require.NotNil(t, params)

	// Test custom params setting
	votePeriod := uint64(10)
	voteThreshold := sdk.NewDecWithPrec(33, 2)
	minVoters := uint64(4)
	oracleRewardBand := sdk.NewDecWithPrec(1, 2)
	slashFraction := sdk.NewDecWithPrec(1, 2)
	slashWindow := uint64(1000)
	minValidPerWindow := sdk.NewDecWithPrec(1, 4)
	minFeeRatio := sdk.NewDecWithPrec(1, 2)
	whitelist := []asset.Pair{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
	}

	// Should really test validateParams, but skipping because obvious
	newParams := types.Params{
		VotePeriod:        votePeriod,
		VoteThreshold:     voteThreshold,
		MinVoters:         minVoters,
		RewardBand:        oracleRewardBand,
		Whitelist:         whitelist,
		SlashFraction:     slashFraction,
		SlashWindow:       slashWindow,
		MinValidPerWindow: minValidPerWindow,
		ValidatorFeeRatio: minFeeRatio,
	}
	input.OracleKeeper.Params.Set(input.Ctx, newParams)

	storedParams, err := input.OracleKeeper.Params.Get(input.Ctx)
	require.NoError(t, err)
	require.NotNil(t, storedParams)
	require.Equal(t, storedParams, newParams)
}

func TestMergeOracleParams(t *testing.T) {
	// baseParams
	votePeriod := uint64(10)

	voteThreshold := sdk.NewDecWithPrec(33, 2)
	changedVoteThreshold := sdk.NewDecWithPrec(50, 2)

	oracleRewardBand := sdk.NewDecWithPrec(1, 2)
	changedRewardBand := sdk.NewDecWithPrec(2, 2)

	whitelist := []asset.Pair{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
	}
	chagedWhitelist := []asset.Pair{
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD),
		asset.Registry.Pair(denoms.ADA, denoms.NUSD),
	}

	slashFraction := sdk.NewDecWithPrec(1, 2)
	changedSlashFraction := sdk.NewDecWithPrec(2, 2)

	slashWindow := uint64(1000)
	changedSlashWindow := uint64(2000)

	minValidPerWindow := sdk.NewDecWithPrec(1, 4)
	changedMinValidPerWindow := sdk.NewDecWithPrec(2, 4)

	twapLoopbackWindow := time.Duration(1000)
	changedTwapLoopbackWindow := time.Duration(2000)

	minVoters := uint64(4)
	chagedMinVoters := uint64(5)

	minFeeRatio := sdk.NewDecWithPrec(1, 2)
	changedMinFeeRatio := sdk.NewDecWithPrec(2, 2)

	expirationBlocks := uint64(100)
	changedExpirationBlocks := uint64(200)

	initialParams := types.Params{
		VotePeriod:         votePeriod,
		VoteThreshold:      voteThreshold,
		MinVoters:          minVoters,
		RewardBand:         oracleRewardBand,
		Whitelist:          whitelist,
		SlashFraction:      slashFraction,
		SlashWindow:        slashWindow,
		MinValidPerWindow:  minValidPerWindow,
		ValidatorFeeRatio:  minFeeRatio,
		TwapLookbackWindow: twapLoopbackWindow,
		ExpirationBlocks:   expirationBlocks,
	}

	tests := []struct {
		name    string
		msg     *types.MsgEditOracleParams
		require func(params types.Params)
	}{
		{
			name: "votePeriod",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					VotePeriod: 20,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, uint64(20), params.VotePeriod)
			},
		},
		{
			name: "votePeriod zero not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					VotePeriod: 0,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, uint64(10), params.VotePeriod)
			},
		},
		{
			name: "voteThreshold",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					VoteThreshold: &changedVoteThreshold,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, changedVoteThreshold, params.VoteThreshold)
			},
		},
		{
			name: "voteThreshold nil not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					VoteThreshold: nil,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, voteThreshold, params.VoteThreshold)
			},
		},
		{
			name: "empty voteThreshold not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					VoteThreshold: &sdk.Dec{},
				},
			},
			require: func(params types.Params) {
				require.Equal(t, voteThreshold, params.VoteThreshold)
			},
		},
		{
			name: "rewardBand",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					RewardBand: &changedRewardBand,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, changedRewardBand, params.RewardBand)
			},
		},
		{
			name: "rewardBand nil not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					RewardBand: nil,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, oracleRewardBand, params.RewardBand)
			},
		},
		{
			name: "empty rewardBand not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					RewardBand: &sdk.Dec{},
				},
			},
			require: func(params types.Params) {
				require.Equal(t, oracleRewardBand, params.RewardBand)
			},
		},
		{
			name: "whitelist",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					Whitelist: chagedWhitelist,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, chagedWhitelist, params.Whitelist)
			},
		},
		{
			name: "whitelist nil not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					Whitelist: nil,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, whitelist, params.Whitelist)
			},
		},
		{
			name: "empty whitelist not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					Whitelist: []asset.Pair{},
				},
			},
			require: func(params types.Params) {
				require.Equal(t, whitelist, params.Whitelist)
			},
		},
		{
			name: "slashFraction",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					SlashFraction: &changedSlashFraction,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, changedSlashFraction, params.SlashFraction)
			},
		},
		{
			name: "slashFraction nil not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					SlashFraction: nil,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, slashFraction, params.SlashFraction)
			},
		},
		{
			name: "empty slashFraction not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					SlashFraction: &sdk.Dec{},
				},
			},
			require: func(params types.Params) {
				require.Equal(t, slashFraction, params.SlashFraction)
			},
		},
		{
			name: "slashWindow",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					SlashWindow: changedSlashWindow,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, changedSlashWindow, params.SlashWindow)
			},
		},
		{
			name: "slashWindow zero not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					SlashWindow: 0,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, slashWindow, params.SlashWindow)
			},
		},
		{
			name: "minValidPerWindow",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					MinValidPerWindow: &changedMinValidPerWindow,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, changedMinValidPerWindow, params.MinValidPerWindow)
			},
		},
		{
			name: "minValidPerWindow nil not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					MinValidPerWindow: nil,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, minValidPerWindow, params.MinValidPerWindow)
			},
		},
		{
			name: "empty minValidPerWindow not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					MinValidPerWindow: &sdk.Dec{},
				},
			},
			require: func(params types.Params) {
				require.Equal(t, minValidPerWindow, params.MinValidPerWindow)
			},
		},
		{
			name: "twapLookbackWindow",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					TwapLookbackWindow: &changedTwapLoopbackWindow,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, changedTwapLoopbackWindow, params.TwapLookbackWindow)
			},
		},
		{
			name: "twapLookbackWindow nil not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					TwapLookbackWindow: nil,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, twapLoopbackWindow, params.TwapLookbackWindow)
			},
		},
		{
			name: "minVoters",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					MinVoters: chagedMinVoters,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, chagedMinVoters, params.MinVoters)
			},
		},
		{
			name: "minVoters zero not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					MinVoters: 0,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, minVoters, params.MinVoters)
			},
		},
		{
			name: "validatorFeeRatio",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					ValidatorFeeRatio: &changedMinFeeRatio,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, changedMinFeeRatio, params.ValidatorFeeRatio)
			},
		},
		{
			name: "validatorFeeRatio nil not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					ValidatorFeeRatio: nil,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, minFeeRatio, params.ValidatorFeeRatio)
			},
		},
		{
			name: "empty validatorFeeRatio not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					ValidatorFeeRatio: &sdk.Dec{},
				},
			},
			require: func(params types.Params) {
				require.Equal(t, minFeeRatio, params.ValidatorFeeRatio)
			},
		},
		{
			name: "expirationTime",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					ExpirationBlocks: changedExpirationBlocks,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, changedExpirationBlocks, params.ExpirationBlocks)
			},
		},
		{
			name: "expirationTime zero not updated",
			msg: &types.MsgEditOracleParams{
				Params: &types.OracleParamsMsg{
					ExpirationBlocks: 0,
				},
			},
			require: func(params types.Params) {
				require.Equal(t, expirationBlocks, params.ExpirationBlocks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			newParams := mergeOracleParams(tt.msg, initialParams)
			tt.require(newParams)
		})
	}
}
