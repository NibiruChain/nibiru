package evmante

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"log"
	"path"
	"reflect"
	"runtime"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

// NewAnteHandlerEvm creates the [sdk.AnteHandler] for Ethereum transactions. An
// Ethereum transaction on Nibiru is an instance of [*evm.MsgEthereumTx],
// sometimes given the alias [evm.Tx].
func NewAnteHandlerEvm(
	options ante.AnteHandlerOptions,
) sdk.AnteHandler {
	steps := []AnteStep{
		AnteStepSetupCtx, // outermost AnteDecorator. AnteStepSetupCtx must be called first
		EthSigVerification,
		AnteStepValidateBasic,
		AnteStepMempoolGasPrice,
		AnteStepBlockGasMeter,
		AnteStepVerifyEthAcc,
		AnteStepCanTransfer,
		AnteStepGasWanted,
		AnteStepDeductGas,
		AnteStepIncrementNonce,
		AnteStepEmitPendingEvent,
		AnteStepFiniteGasLimitForABCIDeliverTx,
	}

	stepNames := make([]string, len(steps))
	for idx, step := range steps {
		stepNames[idx] = shortFuncName(step)
	}

	return sdk.ChainAnteDecorators(
		AnteHandlerEvm{
			EVMKeeper: options.EvmKeeper,
			Opts:      options,
			Steps:     steps,
			StepNames: stepNames,
		},
	)
}

// AnteHandle creates an EVM from the message and calls the BlockContext
// CanTransfer function to see if the address can execute the transaction.
func (handlerGroup AnteHandlerEvm) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	msgEthTx, err := evm.RequireStandardEVMTxMsg(tx)
	if err != nil {
		return ctx, err
	}

	sdb := evmstate.NewSDB(
		ctx,
		handlerGroup.EVMKeeper,
		handlerGroup.TxConfig(ctx, msgEthTx.AsTransaction().Hash()),
	)
	log.Printf(
		"EthState AnteHandle BEGIN:\ntxhash: %s\n{ IsCheckTx %v, IsDeliverTx %v  ReCheckTx%v }",
		msgEthTx.Hash, sdb.Ctx().IsCheckTx(), sdb.IsDeliverTx(), sdb.Ctx().IsReCheckTx())
	sdb.SetCtx(
		sdb.Ctx().
			WithIsEvmTx(true).
			WithEvmTxHash(sdb.TxCfg().TxHash),
	)

	for idx, evmHandler := range handlerGroup.Steps {
		err = evmHandler(
			sdb,
			handlerGroup.EVMKeeper,
			msgEthTx,
			simulate,
			handlerGroup.Opts,
		)
		if err != nil {
			log.Printf("AnteHandlerEvm step %v failed: %s",
				handlerGroup.StepNames[idx], err,
			)
			return ctx, err
		}
		log.Printf("AnteHandlerEvm step %v passed",
			handlerGroup.StepNames[idx],
		)
	}

	log.Printf(
		"EthState AnteHandle END (SUCCESS):\ntxhash: %s\n{ IsCheckTx %v, ReCheckTx %v, IsDeliverTx %v }",
		msgEthTx.Hash, sdb.Ctx().IsCheckTx(), sdb.Ctx().IsReCheckTx(), sdb.IsDeliverTx())
	if evmstate.IsDeliverTx(sdb.Ctx()) {
		sdb.Commit() // Persist
	}
	return sdb.Ctx(), nil
}

// AnteHandlerEvm combines multiple ante handler preflight checks as a single
// EVM state transition. Each of the [AnteStep] functions are performed
// sequentially using the same EVM state db pointer and context(s).
type AnteHandlerEvm struct {
	*EVMKeeper
	Steps     []AnteStep
	StepNames []string
	Opts      AnteOptionsEVM
}

type AnteStep = func(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error)

var _ AnteStep = AnteStepTemplate

func AnteStepTemplate(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	return nil
}

type EVMKeeper = evmstate.Keeper

// shortFuncName parses the function name for the given [AnteStep]. This is
// used for semantically rich logging in the EVM ante handler.
func shortFuncName(fn AnteStep) string {
	pc := reflect.ValueOf(fn).Pointer()
	full := runtime.FuncForPC(pc).Name() // e.g. "github.com/./evmante.AnteStepSetupCtx"
	// strip path prefix; keep last package + symbol
	last := path.Base(strings.ReplaceAll(full, "\\", "/"))
	return last // e.g. "evmante.AnteStepSetupCtx"
}
