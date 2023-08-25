package devgas_test

import (
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/NibiruChain/nibiru/app/codec"
	devgas "github.com/NibiruChain/nibiru/x/devgas/v1"
	devgastypes "github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

func (s *GenesisTestSuite) TestAppModule() {
	s.SetupTest()
	appModule := devgas.NewAppModule(
		s.app.DevGasKeeper,
		s.app.AccountKeeper,
		s.app.GetSubspace(devgastypes.ModuleName),
	)

	s.NotPanics(func() {
		s.T().Log("begin and end block")
		appModule.BeginBlock(s.ctx, abci.RequestBeginBlock{})
		appModule.EndBlock(s.ctx, abci.RequestEndBlock{})

		s.T().Log("AppModule.ExportGenesis")
		cdc := s.app.AppCodec()
		jsonBz := appModule.ExportGenesis(s.ctx, cdc)

		genState := new(devgastypes.GenesisState)
		err := cdc.UnmarshalJSON(jsonBz, genState)
		s.NoError(err)
		s.EqualValues(s.genesis, *genState)

		s.T().Log("AppModuleBasic.ValidateGenesis")
		encCfg := codec.MakeEncodingConfig()
		err = appModule.AppModuleBasic.ValidateGenesis(cdc, encCfg.TxConfig, jsonBz)
		s.NoError(err)

		s.T().Log("CLI commands")
		s.NotNil(appModule.AppModuleBasic.GetTxCmd())
		s.NotNil(appModule.AppModuleBasic.GetQueryCmd())
		s.NotEmpty(appModule.QuerierRoute())
	})
}
