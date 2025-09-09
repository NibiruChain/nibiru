package evmmodule_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmmodule"
)

type Suite struct {
	suite.Suite
}

// TestSuite: Runs all the tests in the suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestAppModule() {
	bapp, ctx := testapp.NewNibiruTestAppAndContext()
	appModule := evmmodule.NewAppModule(
		bapp.EvmKeeper,
		bapp.AccountKeeper,
	)

	s.NotPanics(func() {
		s.T().Log("begin and end block")
		appModule.BeginBlock(ctx, abci.RequestBeginBlock{})
		appModule.EndBlock(ctx, abci.RequestEndBlock{})

		s.T().Log("AppModule.ExportGenesis")
		cdc := bapp.AppCodec()
		jsonBz := appModule.ExportGenesis(ctx, cdc)

		genState := new(evm.GenesisState)
		err := cdc.UnmarshalJSON(jsonBz, genState)
		s.NoError(err)

		s.T().Log("AppModuleBasic.ValidateGenesis")
		encCfg := app.MakeEncodingConfig()
		err = appModule.ValidateGenesis(cdc, encCfg.TxConfig, jsonBz)
		s.NoError(err)

		s.T().Log("CLI commands")
		s.NotNil(appModule.GetTxCmd())
		s.NotNil(appModule.GetQueryCmd())
	})
}
