package sudomodule_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	"github.com/NibiruChain/nibiru/v2/x/sudo/sudomodule"
)

type Suite struct {
	suite.Suite
}

// TestSudoModule: Runs all the tests in the suite.
func TestSudoModule(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestAppModule() {
	bapp, ctx := testapp.NewNibiruTestAppAndContext()
	appModule := sudomodule.NewAppModule(
		bapp.AppCodec(),
		bapp.SudoKeeper,
	)

	s.NotPanics(func() {
		s.T().Log("begin and end block")
		appModule.BeginBlock(ctx, abci.RequestBeginBlock{})
		appModule.EndBlock(ctx, abci.RequestEndBlock{})

		s.T().Log("AppModule.ExportGenesis")
		cdc := bapp.AppCodec()
		jsonBz := appModule.ExportGenesis(ctx, cdc)

		genState := new(sudo.GenesisState)
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
