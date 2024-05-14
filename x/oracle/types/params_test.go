package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestParamsEqual(t *testing.T) {
	p1 := types.DefaultParams()
	err := p1.Validate()
	require.NoError(t, err)

	// minus vote period
	p1.VotePeriod = 0
	err = p1.Validate()
	require.Error(t, err)

	p1.MinVoters = 0
	err = p1.Validate()
	require.Error(t, err)

	// small vote threshold
	p2 := types.DefaultParams()
	p2.VoteThreshold = math.LegacyZeroDec()
	err = p2.Validate()
	require.Error(t, err)

	// negative reward band
	p3 := types.DefaultParams()
	p3.RewardBand = math.LegacyNewDecWithPrec(-1, 2)
	err = p3.Validate()
	require.Error(t, err)

	// negative slash fraction
	p4 := types.DefaultParams()
	p4.SlashFraction = math.LegacyNewDec(-1)
	err = p4.Validate()
	require.Error(t, err)

	// negative min valid per window
	p5 := types.DefaultParams()
	p5.MinValidPerWindow = math.LegacyNewDec(-1)
	err = p5.Validate()
	require.Error(t, err)

	// small slash window
	p6 := types.DefaultParams()
	p6.SlashWindow = 0
	err = p6.Validate()
	require.Error(t, err)

	// empty name
	p10 := types.DefaultParams()
	p10.Whitelist[0] = ""
	err = p10.Validate()
	require.Error(t, err)

	// oracle fee ratio > 1
	p12 := types.DefaultParams()
	p12.ValidatorFeeRatio = math.LegacyNewDec(2)
	err = p12.Validate()
	require.Error(t, err)

	// oracle fee ratio < 0
	p13 := types.DefaultParams()
	p13.ValidatorFeeRatio = math.LegacyNewDec(-1)
	err = p13.Validate()
	require.Error(t, err)

	p11 := types.DefaultParams()
	require.NotNil(t, p11.String())
}
