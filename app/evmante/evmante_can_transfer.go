// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// CanTransferDecorator checks if the sender is allowed to transfer funds according to the EVM block
// context rules.
type CanTransferDecorator struct {
	*EVMKeeper
}

// AnteHandle creates an EVM from the message and calls the BlockContext CanTransfer function to
// see if the address can execute the transaction.
func (ctd CanTransferDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	ethCfg := evm.EthereumConfig(ctd.EthChainID(ctx))
	signer := gethcore.MakeSigner(
		ethCfg,
		big.NewInt(ctx.BlockHeight()),
		evm.ParseBlockTimeUnixU64(ctx),
	)

	msgEthTx, err := evm.RequireStandardEVMTxMsg(tx)
	if err != nil {
		return ctx, err
	}

	baseFeeWeiPerGas := ctd.BaseFeeWeiPerGas(ctx)
	coreMsg, err := msgEthTx.ToGethCoreMsg(signer, baseFeeWeiPerGas)
	if err != nil {
		return ctx, sdkioerrors.Wrapf(
			err,
			"failed to create an ethereum core.Message from signer %T", signer,
		)
	}

	if baseFeeWeiPerGas == nil {
		return ctx, sdkioerrors.Wrap(
			evm.ErrInvalidBaseFee,
			"base fee is nil for this block.",
		)
	}

	if msgEthTx.EffectiveGasCapWei(baseFeeWeiPerGas).Cmp(baseFeeWeiPerGas) < 0 {
		return ctx, sdkioerrors.Wrapf(
			sdkerrors.ErrInsufficientFee,
			"gas fee cap (wei) less than block base fee (wei); (%s < %s)",
			coreMsg.GasFeeCap, baseFeeWeiPerGas,
		)
	}

	// check that caller has enough balance to cover asset transfer for **topmost** call
	// NOTE: here the gas consumed is from the context with the infinite gas meter

	if coreMsg.Value.Sign() > 0 {
		nibiruAddr := eth.EthAddrToNibiruAddr(coreMsg.From)
		balanceWei := ctd.Bank.GetWeiBalance(ctx, nibiruAddr)

		if balanceWei.ToBig().Cmp(coreMsg.Value) < 0 {
			return ctx, sdkioerrors.Wrapf(
				sdkerrors.ErrInsufficientFunds,
				"failed to transfer %s wei ( balance=%s )from address %s using the EVM block context transfer function",
				coreMsg.Value,
				balanceWei,
				coreMsg.From,
			)
		}
	}

	return next(ctx, tx, simulate)
}
