package evmtest

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gethcommon "github.com/ethereum/go-ethereum/common"

	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/codec"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/keeper"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

type TestDeps struct {
	Chain    *app.NibiruApp
	Ctx      sdk.Context
	EncCfg   codec.EncodingConfig
	K        keeper.Keeper
	GenState *evm.GenesisState
	Sender   EthPrivKeyAcc
}

func (deps *TestDeps) GoCtx() context.Context {
	return sdk.WrapSDKContext(deps.Ctx)
}

func NewTestDeps() TestDeps {
	testapp.EnsureNibiruPrefix()
	encCfg := app.MakeEncodingConfig()
	evm.RegisterInterfaces(encCfg.InterfaceRegistry)
	eth.RegisterInterfaces(encCfg.InterfaceRegistry)
	chain, ctx := testapp.NewNibiruTestAppAndContext()
	ctx = ctx.WithChainID(eth.EIP155ChainID_Testnet)
	ethAcc := NewEthAccInfo()
	return TestDeps{
		Chain:    chain,
		Ctx:      ctx,
		EncCfg:   encCfg,
		K:        chain.EvmKeeper,
		GenState: evm.DefaultGenesisState(),
		Sender:   ethAcc,
	}
}

func (deps *TestDeps) StateDB() *statedb.StateDB {
	return statedb.New(deps.Ctx, &deps.Chain.EvmKeeper,
		statedb.NewEmptyTxConfig(
			gethcommon.BytesToHash(deps.Ctx.HeaderHash().Bytes()),
		),
	)
}

func (deps *TestDeps) GethSigner() gethcore.Signer {
	ctx := deps.Ctx
	return deps.Sender.GethSigner(deps.Chain.EvmKeeper.EthChainID(ctx))
}
