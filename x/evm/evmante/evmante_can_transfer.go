package evmante

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

var _ AnteStep = AnteStepVerifyEthAcc

// AnteStepVerifyEthAcc validates checks that the sender balance is greater than the total
// transaction cost. The account will be set to store if it doesn't exist, i.e.
// cannot be found on store.
//
// This AnteHandler decorator will fail if:
// - from address is empty
// - account balance is lower than the transaction cost
func AnteStepVerifyEthAcc(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return sdkioerrors.Wrapf(err, "failed to unpack tx data any for tx %d", 0)
	}

	// sender address should be in the tx cache fromBech32 the previous AnteHandle call
	fromBech32 := msgEthTx.FromAddrBech32()
	if fromBech32.Empty() || len(msgEthTx.From) == 0 {
		return sdkioerrors.Wrap(sdkerrors.ErrInvalidAddress, "from address cannot be empty")
	}

	fromAddr := msgEthTx.FromAddr()

	// Create account if it doesn't exist but has a balance.
	//
	// This is necessary because EVM state transitions (via AddBalance) can create
	// balances in the bank store without creating corresponding accounts in the
	// account store. This creates an inconsistent state where a sender can have
	// a balance (visible to GetBalance) but no account (visible to GetAccount).
	//
	// [AnteStepIncrementNonce] expects the account to exist, so we must create
	// it here to maintain logical consistency: if someone has a balance, they
	// should have an account.
	if acc := k.GetAccount(sdb.Ctx(), fromAddr); acc == nil {
		// Create account if it doesn't exist but has a balance
		emptyAcc := evmstate.NewEmptyAccount()
		if err := k.SetAccount(sdb.Ctx(), fromAddr, *emptyAcc); err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}
	}

	if err := evmstate.CheckSenderBalance(
		sdb.GetBalance(fromAddr), txData,
	); err != nil {
		return err
	}

	return nil
}

var _ AnteStep = AnteStepCanTransfer

func AnteStepCanTransfer(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	baseFeeWeiPerGas := k.BaseFeeWeiPerGas(sdb.Ctx())
	ethCfg := evm.EthereumConfig(k.EthChainID(sdb.Ctx()))
	signer := gethcore.MakeSigner(
		ethCfg,
		big.NewInt(sdb.Ctx().BlockHeight()),
		evm.ParseBlockTimeUnixU64(sdb.Ctx()),
	)
	coreMsg, err := msgEthTx.ToGethCoreMsg(signer, baseFeeWeiPerGas)
	if err != nil {
		return sdkioerrors.Wrapf(
			err,
			"failed to create an ethereum core.Message from signer %T", signer,
		)
	}

	if baseFeeWeiPerGas == nil {
		return sdkioerrors.Wrap(
			evm.ErrInvalidBaseFee,
			"base fee is nil for this block.",
		)
	}

	if msgEthTx.EffectiveGasCapWei(baseFeeWeiPerGas).Cmp(baseFeeWeiPerGas) < 0 {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrInsufficientFee,
			"gas fee cap (wei) less than block base fee (wei); (%s < %s)",
			coreMsg.GasFeeCap, baseFeeWeiPerGas,
		)
	}

	// Check that caller has enough balance to cover asset transfer for the
	// outermost EVM call. Other operations within the EVM state transition might
	// require additional funds inside of the Ethereum tx.
	if coreMsg.Value != nil && coreMsg.Value.Sign() > 0 {
		balanceWei := sdb.GetBalance(coreMsg.From)
		if balanceWei.ToBig().Cmp(coreMsg.Value) < 0 {
			return sdkioerrors.Wrapf(
				sdkerrors.ErrInsufficientFunds,
				"failed to transfer %s wei ( balance=%s ) from address %s using the EVM block context transfer function",
				coreMsg.Value,
				balanceWei,
				coreMsg.From,
			)
		}
	}
	return nil
}
