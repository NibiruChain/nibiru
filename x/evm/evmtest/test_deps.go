package evmtest

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gethcommon "github.com/ethereum/go-ethereum/common"

	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/codec"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

type TestDeps struct {
	App       *app.NibiruApp
	Ctx       sdk.Context
	EncCfg    codec.EncodingConfig
	EvmKeeper *keeper.Keeper
	GenState  *evm.GenesisState
	Sender    EthPrivKeyAcc
}

func NewTestDeps() TestDeps {
	testapp.EnsureNibiruPrefix()
	encCfg := app.MakeEncodingConfig()
	evm.RegisterInterfaces(encCfg.InterfaceRegistry)
	eth.RegisterInterfaces(encCfg.InterfaceRegistry)
	app, ctx := testapp.NewNibiruTestAppAndContext()
	ctx = ctx.WithChainID(eth.EIP155ChainID_Testnet)
	ethAcc := NewEthPrivAcc()
	return TestDeps{
		App:       app,
		Ctx:       ctx,
		EncCfg:    encCfg,
		EvmKeeper: app.EvmKeeper,
		GenState:  evm.DefaultGenesisState(),
		Sender:    ethAcc,
	}
}

func (deps TestDeps) NewStateDB() *statedb.StateDB {
	return deps.EvmKeeper.NewStateDB(
		deps.Ctx,
		statedb.NewEmptyTxConfig(
			gethcommon.BytesToHash(deps.Ctx.HeaderHash().Bytes()),
		),
	)
}

func (deps *TestDeps) GethSigner() gethcore.Signer {
	return gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx))
}

func (deps TestDeps) GoCtx() context.Context {
	return sdk.WrapSDKContext(deps.Ctx)
}

func (deps TestDeps) ResetGasMeter() {
	deps.EvmKeeper.ResetTransientGasUsed(deps.Ctx)
	deps.EvmKeeper.ResetGasMeterAndConsumeGas(deps.Ctx, 0)
}
