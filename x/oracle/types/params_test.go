package types_test

import (
	"bytes"
	"testing"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
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
	p2.VoteThreshold = sdk.ZeroDec()
	err = p2.Validate()
	require.Error(t, err)

	// negative reward band
	p3 := types.DefaultParams()
	p3.RewardBand = sdk.NewDecWithPrec(-1, 2)
	err = p3.Validate()
	require.Error(t, err)

	// negative slash fraction
	p4 := types.DefaultParams()
	p4.SlashFraction = sdk.NewDec(-1)
	err = p4.Validate()
	require.Error(t, err)

	// negative min valid per window
	p5 := types.DefaultParams()
	p5.MinValidPerWindow = sdk.NewDec(-1)
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

	p11 := types.DefaultParams()
	require.NotNil(t, p11.ParamSetPairs())
	require.NotNil(t, p11.String())
}

func TestValidate(t *testing.T) {
	p1 := types.DefaultParams()
	pairs := p1.ParamSetPairs()
	for _, pair := range pairs {
		switch {
		case bytes.Equal(types.KeyVotePeriod, pair.Key) ||
			bytes.Equal(types.KeySlashWindow, pair.Key) ||
			bytes.Equal(types.KeyMinVoters, pair.Key):
			require.NoError(t, pair.ValidatorFn(uint64(1)))
			require.Error(t, pair.ValidatorFn("invalid"))
			require.Error(t, pair.ValidatorFn(uint64(0)))
		case bytes.Equal(types.KeyVoteThreshold, pair.Key):
			require.NoError(t, pair.ValidatorFn(sdk.NewDecWithPrec(33, 2)))
			require.Error(t, pair.ValidatorFn("invalid"))
			require.Error(t, pair.ValidatorFn(sdk.NewDecWithPrec(32, 2)))
			require.Error(t, pair.ValidatorFn(sdk.NewDecWithPrec(101, 2)))
		case bytes.Equal(types.KeyRewardBand, pair.Key) ||
			bytes.Equal(types.KeySlashFraction, pair.Key) ||
			bytes.Equal(types.KeyMinValidPerWindow, pair.Key):
			require.NoError(t, pair.ValidatorFn(sdk.NewDecWithPrec(7, 2)))
			require.Error(t, pair.ValidatorFn("invalid"))
			require.Error(t, pair.ValidatorFn(sdk.NewDecWithPrec(-1, 2)))
			require.Error(t, pair.ValidatorFn(sdk.NewDecWithPrec(101, 2)))
		case bytes.Equal(types.KeyWhitelist, pair.Key):
			require.NoError(t, pair.ValidatorFn([]asset.Pair{"BTC:USDT"}))
			require.Error(t, pair.ValidatorFn("invalid"))
			require.Error(t, pair.ValidatorFn([]asset.Pair{""}))
		}
	}
}
