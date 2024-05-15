package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

func (s *TestSuite) TestGenesis() {
	// Produces a valid token factory denom
	randomTFDenom := func() string {
		denom := types.TFDenom{
			Creator:  testutil.AccAddress().String(),
			Subdenom: testutil.RandLetters(3),
		}
		s.Require().NoError(denom.Validate())
		return denom.Denom().String()
	}

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
			name: "genesis populated with valid data",
			genesis: types.GenesisState{
				Params: types.DefaultModuleParams(),
				FactoryDenoms: []types.GenesisDenom{
					{
						Denom: randomTFDenom(),
						AuthorityMetadata: types.DenomAuthorityMetadata{
							Admin: testutil.AccAddress().String(),
						},
					},
					{
						Denom: randomTFDenom(),
						AuthorityMetadata: types.DenomAuthorityMetadata{
							Admin: testutil.AccAddress().String(),
						},
					},
				},
			},
			expPanic: false,
		},
		// {}, // Invalid test case
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			if tc.expPanic {
				s.Require().Panics(func() {
				})
			} else {
				s.Require().NotPanics(func() {
					s.app.TokenFactoryKeeper.InitGenesis(s.ctx, tc.genesis)
				})

				params, err := s.app.TokenFactoryKeeper.Store.
					ModuleParams.Get(s.ctx)
				s.NoError(err)
				s.Require().EqualValues(tc.genesis.Params, params)

				gen := s.app.TokenFactoryKeeper.ExportGenesis(s.ctx)
				s.NoError(gen.Validate())
			}
		})
	}
}
