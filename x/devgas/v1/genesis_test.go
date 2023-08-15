package devgas_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/devgas/v1"
	devgastypes "github.com/NibiruChain/nibiru/x/devgas/v1/types"
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

func (suite *GenesisTestSuite) SetupTest() {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	suite.app = app
	suite.ctx = ctx

	suite.genesis = *devgastypes.DefaultGenesisState()
}

func (suite *GenesisTestSuite) TestFeeShareInitGenesis() {
	testCases := []struct {
		name     string
		genesis  devgastypes.GenesisState
		expPanic bool
	}{
		{
			"default genesis",
			suite.genesis,
			false,
		},
		{
			"custom genesis - feeshare disabled",
			devgastypes.GenesisState{
				Params: devgastypes.Params{
					EnableFeeShare:  false,
					DeveloperShares: devgastypes.DefaultDeveloperShares,
					AllowedDenoms:   []string{"ujuno"},
				},
			},
			false,
		},
		{
			"custom genesis - feeshare enabled, 0% developer shares",
			devgastypes.GenesisState{
				Params: devgastypes.Params{
					EnableFeeShare:  true,
					DeveloperShares: sdk.NewDecWithPrec(0, 2),
					AllowedDenoms:   []string{"ujuno"},
				},
			},
			false,
		},
		{
			"custom genesis - feeshare enabled, 100% developer shares",
			devgastypes.GenesisState{
				Params: devgastypes.Params{
					EnableFeeShare:  true,
					DeveloperShares: sdk.NewDecWithPrec(100, 2),
					AllowedDenoms:   []string{"ujuno"},
				},
			},
			false,
		},
		{
			"custom genesis - feeshare enabled, all denoms allowed",
			devgastypes.GenesisState{
				Params: devgastypes.Params{
					EnableFeeShare:  true,
					DeveloperShares: sdk.NewDecWithPrec(10, 2),
					AllowedDenoms:   []string(nil),
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			if tc.expPanic {
				suite.Require().Panics(func() {
					devgas.InitGenesis(suite.ctx, suite.app.DevGasKeeper, tc.genesis)
				})
			} else {
				suite.Require().NotPanics(func() {
					devgas.InitGenesis(suite.ctx, suite.app.DevGasKeeper, tc.genesis)
				})

				params := suite.app.DevGasKeeper.GetParams(suite.ctx)
				suite.Require().Equal(tc.genesis.Params, params)
			}
		})
	}
}
