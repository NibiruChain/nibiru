package tokenfactory_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app/codec"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	module "github.com/NibiruChain/nibiru/x/tokenfactory"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
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
		encCfg := codec.MakeEncodingConfig()
		err = appModule.AppModuleBasic.ValidateGenesis(cdc, encCfg.TxConfig, jsonBz)
		s.NoError(err)

		s.T().Log("CLI commands")
		// TODO
		// s.NotNil(appModule.AppModuleBasic.GetTxCmd())
		// s.NotNil(appModule.AppModuleBasic.GetQueryCmd())
		s.NotEmpty(appModule.QuerierRoute())
	})
}
