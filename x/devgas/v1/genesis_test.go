package devgas_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	devgas "github.com/NibiruChain/nibiru/v2/x/devgas/v1"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
)

type GenesisTestSuite struct {
	suite.Suite

	app *app.NibiruApp
	ctx sdk.Context

	genesis devgastypes.GenesisState
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (s *GenesisTestSuite) SetupTest() {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	s.app = app
	s.ctx = ctx

	s.genesis = *devgastypes.DefaultGenesisState()
}

func (s *GenesisTestSuite) TestGenesis() {
	randomAddr := testutil.AccAddress().String()
	testCases := []struct {
		name     string
		genesis  devgastypes.GenesisState
		expPanic bool
	}{
		{
			name:     "default genesis",
			genesis:  s.genesis,
			expPanic: false,
		},
		{
			name: "custom genesis - feeshare disabled",
			genesis: devgastypes.GenesisState{
				Params: devgastypes.ModuleParams{
					EnableFeeShare:  false,
					DeveloperShares: devgastypes.DefaultDeveloperShares,
					AllowedDenoms:   []string{"unibi"},
				},
			},
			expPanic: false,
		},
		{
			name: "custom genesis - feeshare enabled, 0% developer shares",
			genesis: devgastypes.GenesisState{
				Params: devgastypes.ModuleParams{
					EnableFeeShare:  true,
					DeveloperShares: math.LegacyNewDecWithPrec(0, 2),
					AllowedDenoms:   []string{"unibi"},
				},
			},
			expPanic: false,
		},
		{
			name: "custom genesis - feeshare enabled, 100% developer shares",
			genesis: devgastypes.GenesisState{
				Params: devgastypes.ModuleParams{
					EnableFeeShare:  true,
					DeveloperShares: math.LegacyNewDecWithPrec(100, 2),
					AllowedDenoms:   []string{"unibi"},
				},
			},
			expPanic: false,
		},
		{
			name: "custom genesis - feeshare enabled, all denoms allowed",
			genesis: devgastypes.GenesisState{
				Params: devgastypes.ModuleParams{
					EnableFeeShare:  true,
					DeveloperShares: math.LegacyNewDecWithPrec(10, 2),
					AllowedDenoms:   []string{},
				},
				FeeShare: []devgastypes.FeeShare{
					{
						ContractAddress:   randomAddr,
						DeployerAddress:   randomAddr,
						WithdrawerAddress: randomAddr,
					},
				},
			},
			expPanic: false,
		},
		{
			name:     "empty genesis",
			genesis:  devgastypes.GenesisState{},
			expPanic: true,
		},
	}

	for _, tc := range testCases {
		s.T().Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			s.SetupTest() // reset

			if tc.expPanic {
				s.Require().Panics(func() {
					devgas.InitGenesis(s.ctx, s.app.DevGasKeeper, tc.genesis)
				})
			} else {
				s.Require().NotPanics(func() {
					devgas.InitGenesis(s.ctx, s.app.DevGasKeeper, tc.genesis)
				})

				params := s.app.DevGasKeeper.GetParams(s.ctx)
				s.Require().EqualValues(tc.genesis.Params, params)

				gen := devgas.ExportGenesis(s.ctx, s.app.DevGasKeeper)
				s.NoError(gen.Validate())
			}
		})
	}
}
