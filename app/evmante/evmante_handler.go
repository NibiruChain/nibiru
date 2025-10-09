// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
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
				// TODO: UD-DEBUG: Handlers to impl
				EthAnteVerifyEthAcc,
				EthAnteCanTransfer,
				EthAnteGasConsume,
				EthAnteIncrementNonce,
			},
		},
		// NewEthSigVerificationDecorator(options.EvmKeeper),
		// NewAnteDecVerifyEthAcc(options.EvmKeeper, options.AccountKeeper),
		// CanTransferDecorator{options.EvmKeeper},
		// NewAnteDecEthGasConsume(options.EvmKeeper, options.MaxTxGasWanted),
		NewAnteDecEthIncrementSenderSequence(options.EvmKeeper, options.AccountKeeper),
		ante.AnteDecoratorGasWanted{},
		// emit eth tx hash and index at the very last ante handler.
		NewEthEmitEventDecorator(options.EvmKeeper),
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
	sdb.SetCtx(
		sdb.Ctx().
			WithIsEvmTx(true).
			WithEvmTxHash(sdb.TxCfg().TxHash),
	)
	for _, evmHandler := range handlerGroup.Body {
		err = evmHandler(
			sdb,
			handlerGroup.EVMKeeper,
			msgEthTx,
			simulate,
			handlerGroup.Opts,
		)
		if err != nil {
			return ctx, err
		}

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
