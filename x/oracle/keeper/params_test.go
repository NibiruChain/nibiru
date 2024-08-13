package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
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
	voteThreshold := math.LegacyNewDecWithPrec(33, 2)
	minVoters := uint64(4)
	oracleRewardBand := math.LegacyNewDecWithPrec(1, 2)
	slashFraction := math.LegacyNewDecWithPrec(1, 2)
	slashWindow := uint64(1000)
	minValidPerWindow := math.LegacyNewDecWithPrec(1, 4)
	minFeeRatio := math.LegacyNewDecWithPrec(1, 2)
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
