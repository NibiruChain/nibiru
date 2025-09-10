package evmtest

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	core "github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

type TestDeps struct {
	App       *app.NibiruApp
	Ctx       sdk.Context
	EvmKeeper *keeper.Keeper
	GenState  *evm.GenesisState
	Sender    EthPrivKeyAcc
}

func NewTestDeps() TestDeps {
	app, ctx := testapp.NewNibiruTestAppAndContext()
	ctx = ctx.WithChainID(eth.EIP155ChainID_Testnet)

	return TestDeps{
		App:       app,
		Ctx:       ctx,
		EvmKeeper: app.EvmKeeper,
		GenState:  evm.DefaultGenesisState(),
		Sender:    NewEthPrivAcc(),
	}
}

func (deps TestDeps) NewStateDB() *statedb.StateDB {
	return deps.EvmKeeper.NewStateDB(
		deps.Ctx,
		statedb.NewEmptyTxConfig(
			gethcommon.BytesToHash(deps.Ctx.HeaderHash()),
		),
	)
}

func (deps TestDeps) NewEVM() (*vm.EVM, *statedb.StateDB) {
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, statedb.NewEmptyTxConfig(gethcommon.BytesToHash(deps.Ctx.HeaderHash())))
	evmObj := deps.EvmKeeper.NewEVM(
		deps.Ctx,
		MOCK_GETH_MESSAGE,
		deps.EvmKeeper.GetEVMConfig(deps.Ctx),
		logger.NewStructLogger(&logger.Config{Debug: false}).Hooks(),
		stateDB,
	)
	return evmObj, stateDB
}

func (deps TestDeps) NewEVMLessVerboseLogger() (*vm.EVM, *statedb.StateDB) {
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, statedb.NewEmptyTxConfig(gethcommon.BytesToHash(deps.Ctx.HeaderHash())))
	evmObj := deps.EvmKeeper.NewEVM(
		deps.Ctx,
		MOCK_GETH_MESSAGE,
		deps.EvmKeeper.GetEVMConfig(deps.Ctx),
		logger.NewStructLogger(&logger.Config{Debug: false}).Hooks(),
		stateDB,
	)
	return evmObj, stateDB
}

func (deps *TestDeps) GethSigner() gethcore.Signer {
	return gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx))
}

func (deps TestDeps) GoCtx() context.Context {
	return sdk.WrapSDKContext(deps.Ctx)
}

func (deps *TestDeps) MimicEthereumTx(
	s *suite.Suite,
	doTx func(evmObj *vm.EVM, sdb *statedb.StateDB),
) {
	sdb := deps.EvmKeeper.NewStateDB(
		deps.Ctx,
		statedb.NewEmptyTxConfig(gethcommon.BytesToHash(deps.Ctx.HeaderHash())),
	)
	evmObj := deps.EvmKeeper.NewEVM(
		deps.Ctx,
		MOCK_GETH_MESSAGE,
		deps.EvmKeeper.GetEVMConfig(deps.Ctx),
		logger.NewStructLogger(&logger.Config{Debug: false}).Hooks(),
		sdb,
	)
	doTx(evmObj, sdb)
	s.Require().NoError(sdb.Commit())
}

func (deps *TestDeps) DeployWNIBI(s *suite.Suite) {
	var (
		ctx         = deps.Ctx
		wnibiAddr   = deps.EvmKeeper.GetParams(ctx).CanonicalWnibi.Address
		evmAccState = deps.EvmKeeper.EvmState.AccState
	)

	evmModuleNonce := deps.EvmKeeper.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS)
	tempWnibiAddr := crypto.CreateAddress(evm.EVM_MODULE_ADDRESS, evmModuleNonce)
	newCompiledContract := embeds.SmartContract_WNIBI
	// empty method name means deploy with the constructor
	packedArgs, err := newCompiledContract.ABI.Pack("")
	s.Require().NoError(err, "failed to pack ABI args")

	contractInput := append(newCompiledContract.Bytecode, packedArgs...)

	// Rebuild evmObj with new evmMsg for contract creation.
	// Note that most of these fields are unused when we create EVM instances
	// outside of an EthereumTx.
	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               nil,                    // To is blank -> deploy contract
		From:             evm.EVM_MODULE_ADDRESS, // From is the deployer
		Nonce:            evmModuleNonce,
		Value:            unusedBigInt, // amount
		GasLimit:         keeper.Erc20GasLimitDeploy,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             contractInput, // This manages the constructor args
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	stateDB := deps.EvmKeeper.Bank.StateDB
	if stateDB == nil {
		stateDB = deps.EvmKeeper.NewStateDB(ctx, deps.EvmKeeper.TxConfig(ctx, gethcommon.Hash{}))
	}
	defer func() {
		deps.EvmKeeper.Bank.StateDB = nil
	}()
	evmObj := deps.EvmKeeper.NewEVM(ctx, evmMsg, deps.EvmKeeper.GetEVMConfig(ctx), nil, stateDB)

	evmResp, err := deps.EvmKeeper.CallContract(
		ctx, evmObj, evmMsg.From, nil, contractInput,
		keeper.Erc20GasLimitDeploy,
		evm.COMMIT_ETH_TX, /*commit*/
		evmMsg.Value,
	)
	s.Require().NoError(err, "failed to deploy WNIBI contract")
	s.Require().Empty(evmResp.VmError, "VM Error deploying WNIBI")

	_ = ctx.EventManager().EmitTypedEvents(
		&evm.EventContractDeployed{
			Sender:       evmMsg.From.Hex(),
			ContractAddr: tempWnibiAddr.Hex(),
		},
	)

	s.T().Logf("Set WNIBI bytecode hash at address %s", wnibiAddr)
	tempWnibiAcc := deps.EvmKeeper.GetAccount(ctx, tempWnibiAddr)
	wnibiAcc := statedb.NewEmptyAccount()
	wnibiAcc.CodeHash = tempWnibiAcc.CodeHash
	err = deps.EvmKeeper.SetAccount(ctx, wnibiAddr, *wnibiAcc)
	s.Require().NoError(err, "overwrite of contract bytecode failed")

	s.T().Log("Set WNIBI contract state")
	{
		iter := evmAccState.Iterate(ctx, collections.PairRange[gethcommon.Address, gethcommon.Hash]{}.Prefix(tempWnibiAddr))
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			evmAccState.Insert(
				ctx,
				collections.Join(wnibiAddr, iter.Key().K2()),
				iter.Value(),
			)
		}
	}
	_ = ctx.EventManager().EmitTypedEvents(
		&evm.EventContractDeployed{
			Sender:       evmMsg.From.Hex(),
			ContractAddr: wnibiAddr.Hex(),
		},
	)
}
