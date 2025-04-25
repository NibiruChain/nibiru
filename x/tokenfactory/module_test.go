package tokenfactory_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	module "github.com/NibiruChain/nibiru/v2/x/tokenfactory"
	"github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

type ModuleTestSuite struct{ suite.Suite }

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

func (s *ModuleTestSuite) TestAppModule() {
	bapp, ctx := testapp.NewNibiruTestAppAndContext()
	appModule := module.NewAppModule(
		bapp.TokenFactoryKeeper,
		bapp.AccountKeeper,
	)

	s.NotPanics(func() {
		s.T().Log("begin and end block")
		appModule.BeginBlock(ctx)
		appModule.EndBlock(ctx)

		s.T().Log("AppModule.ExportGenesis")
		cdc := bapp.AppCodec()
		jsonBz := appModule.ExportGenesis(ctx, cdc)

		genesis := types.DefaultGenesis()
		genState := new(types.GenesisState)
		err := cdc.UnmarshalJSON(jsonBz, genState)
		s.NoError(err)
		s.EqualValues(*genesis, *genState, "exported (got): %s", jsonBz)

		s.T().Log("AppModuleBasic.ValidateGenesis")
		encCfg := app.MakeEncodingConfig()
		err = appModule.AppModuleBasic.ValidateGenesis(cdc, encCfg.TxConfig, jsonBz)
		s.NoError(err)

		s.T().Log("CLI commands")
		s.NotNil(appModule.AppModuleBasic.GetTxCmd())
		s.NotNil(appModule.AppModuleBasic.GetQueryCmd())
		s.NotEmpty(appModule.QuerierRoute())
	})
}
