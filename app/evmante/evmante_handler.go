// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

// NewAnteHandlerEVM creates the default ante handler for Ethereum transactions
func NewAnteHandlerEVM(
	options ante.AnteHandlerOptions,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// outermost AnteDecorator. SetUpContext must be called first
		NewEthSetUpContextDecorator(options.EvmKeeper),
		NewMempoolGasPriceDecorator(options.EvmKeeper),
		NewEthValidateBasicDecorator(options.EvmKeeper),
		NewEthStateHandlers{
			EVMKeeper: options.EvmKeeper,
			Opts:      options,
			Body: []EvmAnteHandler{
				EthSigVerification,
				EthAnteBlockGasMeter,
				EthAnteVerifyEthAcc,
				EthAnteCanTransfer,
				EthAnteGasWanted,
				EthAnteDeductGas,
				EthAnteIncrementNonce,
				EthAnteFiniteGasLimitForABCIDeliverTx,
				EthAnteEmitPendingEvent,
			},
		},
		// NewAnteDecVerifyEthAcc(options.EvmKeeper, options.AccountKeeper),
		// CanTransferDecorator{options.EvmKeeper},
		// NewAnteDecEthGasConsume(options.EvmKeeper, options.MaxTxGasWanted),
		// NewAnteDecEthIncrementSenderSequence(options.EvmKeeper, options.AccountKeeper),
		// ante.AnteDecBlockGasWanted{},
		// emit eth tx hash and index at the very last ante handler.
		// NewEthEmitEventDecorator(options.EvmKeeper),
	)
}

// AnteHandle creates an EVM from the message and calls the BlockContext
// CanTransfer function to see if the address can execute the transaction.
func (handlerGroup NewEthStateHandlers) AnteHandle(
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
		"EthState AnteHandle BEGIN: %s\n{ IsCheckTx %v, ReCheckTx%v }",
		msgEthTx.Hash, sdb.Ctx().IsCheckTx(), sdb.Ctx().IsReCheckTx())
	sdb.SetCtx(
		sdb.Ctx().
			WithIsEvmTx(true).
			WithEvmTxHash(sdb.TxCfg().TxHash),
	)
	for idx, evmHandler := range handlerGroup.Body {
		err = evmHandler(
			sdb,
			handlerGroup.EVMKeeper,
			msgEthTx,
			simulate,
			handlerGroup.Opts,
		)
		if err != nil {
			log.Printf("EthState AnteHandle Body elem %d failed: %s", idx, err)
			return ctx, err
		}
		log.Printf("EthState AnteHandle Body elem %d passed", idx)
	}

	log.Printf(
		"EthState AnteHandle END (SUCCESS):\ntxhash: %s\n{ IsCheckTx %v, ReCheckTx %v, IsDeliverTx %v }",
		msgEthTx.Hash, sdb.Ctx().IsCheckTx(), sdb.Ctx().IsReCheckTx(), sdb.IsDeliverTx())
	if evmstate.IsDeliverTx(sdb.Ctx()) {
		sdb.Commit() // Persist
	}
	return sdb.Ctx(), nil
}

// NewEthStateHandlers combines multiple ante handler preflight checks as a single
// EVM state transition. Each of the [EvmAnteHandler] functions are performed
// sequentially using the same EVM state db pointer and context(s).
type NewEthStateHandlers struct {
	*EVMKeeper
	Opts AnteOptionsEVM
	Body []EvmAnteHandler
}

type EvmAnteHandler = func(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error)

var _ EvmAnteHandler = EthAnteTemplate

func EthAnteTemplate(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	return nil
}
