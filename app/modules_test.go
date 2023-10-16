package app_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
)

type TestSuite struct {
	suite.Suite

	encCfg app.EncodingConfig
}

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) SetupSuite() {
	s.encCfg = app.MakeEncodingConfig()
}

func (s *TestSuite) DefaultGenesisCopy() app.GenesisState {
	return app.NewDefaultGenesisState(s.encCfg.Marshaler)
}

func (s *TestSuite) TestGenesis() {
	getDefaultStakingGenesis := func() *stakingtypes.GenesisState {
		genStaking := new(stakingtypes.GenesisState)
		s.encCfg.Marshaler.MustUnmarshalJSON(
			app.StakingModule{}.DefaultGenesis(s.encCfg.Marshaler),
			genStaking,
		)
		return genStaking
	}

	gens := []*stakingtypes.GenesisState{}
	gens = append(gens, getDefaultStakingGenesis())

	genStaking := getDefaultStakingGenesis()
	genStaking.Params.MinCommissionRate = sdk.ZeroDec()
	gens = append(gens, genStaking)

	for _, tc := range []struct {
		name    string
		gen     *stakingtypes.GenesisState
		wantErr string
	}{
		{
			name: "default should work fine",
			gen:  gens[0],
		},
		{
			name:    "zero commission should fail",
			gen:     gens[1],
			wantErr: "min_commission must be positive",
		},
	} {
		s.T().Run(tc.name, func(t *testing.T) {
			genStakingJson := s.encCfg.Marshaler.MustMarshalJSON(tc.gen)
			err := app.StakingModule{}.ValidateGenesis(
				s.encCfg.Marshaler,
				s.encCfg.TxConfig,
				genStakingJson,
			)
			if tc.wantErr != "" {
				s.ErrorContains(err, tc.wantErr)
				return
			}
			s.NoError(err)
		})
	}
}
