package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	devgas "github.com/NibiruChain/nibiru/v2/x/devgas/v1"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
)

type GenesisTestSuite struct {
	suite.Suite

	app *app.NibiruApp
	ctx sdk.Context

	genesis types.GenesisState
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (s *GenesisTestSuite) SetupTest() {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	s.app = app
	s.ctx = ctx

	s.genesis = *types.DefaultGenesis()
}

func (s *GenesisTestSuite) TestGenesis() {
	testCases := []struct {
		name     string
		genesis  types.GenesisState
		expPanic bool
	}{
		{
			name:     "default genesis",
			genesis:  s.genesis,
			expPanic: false,
		},
		{
			name: "custom genesis - fee tokens",
			genesis: types.GenesisState{
				Feetokens: validFeeTokens,
			},
			expPanic: false,
		},
		{
			name: "empty genesis",
			genesis: types.GenesisState{
				Feetokens: []types.FeeToken{},
			},
			expPanic: false,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset

			if tc.expPanic {
				s.Require().Panics(func() {
					s.app.GasTokenKeeper.InitGenesis(s.ctx, tc.genesis, s.app.AccountKeeper)
				})
			} else {
				s.Require().NotPanics(func() {
					s.app.GasTokenKeeper.InitGenesis(s.ctx, tc.genesis, s.app.AccountKeeper)
				})

				feeTokens := s.app.GasTokenKeeper.GetFeeTokens(s.ctx)
				sortFeeTokens(feeTokens)
				sortFeeTokens(tc.genesis.Feetokens)
				s.Require().Equal(tc.genesis.Feetokens, feeTokens)

				gen := devgas.ExportGenesis(s.ctx, s.app.DevGasKeeper)
				s.NoError(gen.Validate())
			}
		})
	}
}
