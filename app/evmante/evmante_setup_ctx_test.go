package evmante_test

import (
	"math"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *TestSuite) TestEthSetupContextDecorator() {
	deps := evmtest.NewTestDeps()
	stateDB := deps.NewStateDB()
	anteDec := evmante.NewEthSetUpContextDecorator(deps.App.EvmKeeper)

	s.Require().NoError(stateDB.Commit())
	tx := evmtest.HappyCreateContractTx(&deps)

	// Ante handler returns new context
	newCtx, err := anteDec.AnteHandle(
		deps.Ctx, tx, false, evmtest.NextNoOpAnteHandler,
	)
	s.Require().NoError(err)

	// Check that ctx gas meter is set up to infinite
	ctxGasMeter := newCtx.GasMeter()
	s.Require().Equal(sdk.Gas(math.MaxUint64), ctxGasMeter.GasRemaining())

	// Check that gas configs are reset to default
	defaultGasConfig := storetypes.GasConfig{}
	s.Require().Equal(defaultGasConfig, newCtx.KVGasConfig())
	s.Require().Equal(defaultGasConfig, newCtx.TransientKVGasConfig())
}
