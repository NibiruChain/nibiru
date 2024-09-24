// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"errors"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// EthValidateBasicDecorator is adapted from ValidateBasicDecorator from cosmos-sdk, it ignores ErrNoSignatures
type EthValidateBasicDecorator struct {
	evmKeeper EVMKeeper
}

// NewEthValidateBasicDecorator creates a new EthValidateBasicDecorator
func NewEthValidateBasicDecorator(k EVMKeeper) EthValidateBasicDecorator {
	return EthValidateBasicDecorator{
		evmKeeper: k,
	}
}

// AnteHandle handles basic validation of tx
func (vbd EthValidateBasicDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// no need to validate basic on recheck tx, call next antehandler
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	err := tx.ValidateBasic()
	// ErrNoSignatures is fine with eth tx
	if err != nil && !errors.Is(err, sdkerrors.ErrNoSignatures) {
		return ctx, errorsmod.Wrap(err, "tx basic validation failed")
	}

	// For eth type cosmos tx, some fields should be verified as zero values,
	// since we will only verify the signature against the hash of the MsgEthereumTx.Data
	wrapperTx, ok := tx.(protoTxProvider)
	if !ok {
		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrUnknownRequest,
			"invalid tx type %T, didn't implement interface protoTxProvider",
			tx,
		)
	}

	protoTx := wrapperTx.GetProtoTx()
	body := protoTx.Body
	if body.Memo != "" || body.TimeoutHeight != uint64(0) || len(body.NonCriticalExtensionOptions) > 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest,
			"for eth tx body Memo TimeoutHeight NonCriticalExtensionOptions should be empty")
	}

	if len(body.ExtensionOptions) != 1 {
		return ctx, errorsmod.Wrap(
			sdkerrors.ErrInvalidRequest,
			"for eth tx length of ExtensionOptions should be 1",
		)
	}

	authInfo := protoTx.AuthInfo
	if len(authInfo.SignerInfos) > 0 {
		return ctx, errorsmod.Wrap(
			sdkerrors.ErrInvalidRequest,
			"for eth tx AuthInfo SignerInfos should be empty",
		)
	}

	if authInfo.Fee.Payer != "" || authInfo.Fee.Granter != "" {
		return ctx, errorsmod.Wrap(
			sdkerrors.ErrInvalidRequest,
			"for eth tx AuthInfo Fee payer and granter should be empty",
		)
	}

	sigs := protoTx.Signatures
	if len(sigs) > 0 {
		return ctx, errorsmod.Wrap(
			sdkerrors.ErrInvalidRequest,
			"for eth tx Signatures should be empty",
		)
	}

	txFee := sdk.Coins{}
	txGasLimit := uint64(0)

	evmParams := vbd.evmKeeper.GetParams(ctx)
	baseFee := vbd.evmKeeper.GetBaseFee(ctx)
	evmDenom := evmParams.GetEvmDenom()

	for _, msg := range protoTx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errorsmod.Wrapf(
				sdkerrors.ErrUnknownRequest,
				"invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil),
			)
		}

		// Validate `From` field
		if msgEthTx.From != "" {
			return ctx, errorsmod.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"invalid From %s, expect empty string", msgEthTx.From,
			)
		}

		txGasLimit += msgEthTx.GetGas()

		txData, err := evm.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, errorsmod.Wrap(err, "failed to unpack MsgEthereumTx Data")
		}

		if baseFee == nil && txData.TxType() == gethcore.DynamicFeeTxType {
			return ctx, errorsmod.Wrap(
				gethcore.ErrTxTypeNotSupported,
				"dynamic fee tx not supported",
			)
		}

		txFee = txFee.Add(
			sdk.Coin{
				Denom:  evmDenom,
				Amount: sdkmath.NewIntFromBigInt(txData.Fee()),
			},
		)
	}

	if !authInfo.Fee.Amount.IsEqual(txFee) {
		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"invalid AuthInfo Fee Amount (%s != %s)",
			authInfo.Fee.Amount,
			txFee,
		)
	}

	if authInfo.Fee.GasLimit != txGasLimit {
		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"invalid AuthInfo Fee GasLimit (%d != %d)",
			authInfo.Fee.GasLimit,
			txGasLimit,
		)
	}

	return next(ctx, tx, simulate)
}
