package tokenfactory_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	module "github.com/NibiruChain/nibiru/v2/x/tokenfactory"
	"github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

type Suite struct{ suite.Suite }

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestAppModule() {
	bapp, ctx := testapp.NewNibiruTestAppAndContext()
	appModule := module.NewAppModule(
		bapp.TokenFactoryKeeper,
		bapp.AccountKeeper,
	)

	s.NotPanics(func() {
		s.T().Log("begin and end block")
		appModule.BeginBlock(ctx, abci.RequestBeginBlock{})
		appModule.EndBlock(ctx, abci.RequestEndBlock{})

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
		err = appModule.ValidateGenesis(cdc, encCfg.TxConfig, jsonBz)
		s.NoError(err)

		s.T().Log("CLI commands")
		s.NotNil(appModule.GetTxCmd())
		s.NotNil(appModule.GetQueryCmd())
		s.NotEmpty(appModule.QuerierRoute())
	})
}
