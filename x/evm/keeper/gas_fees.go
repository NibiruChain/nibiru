// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/x/evm"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// GetEthIntrinsicGas returns the intrinsic gas cost for the transaction
func (k *Keeper) GetEthIntrinsicGas(
	ctx sdk.Context,
	msg core.Message,
	cfg *params.ChainConfig,
	isContractCreation bool,
) (uint64, error) {
	return core.IntrinsicGas(
		msg.Data(), msg.AccessList(),
		isContractCreation, true, true,
	)
}

// RefundGas transfers the leftover gas to the sender of the message.
func (k *Keeper) RefundGas(
	ctx sdk.Context,
	msgFrom gethcommon.Address,
	leftoverGas uint64,
	weiPerGas *big.Int,
) error {
	// Return EVM tokens for remaining gas, exchanged at the original rate.
	leftoverWei := new(big.Int).Mul(
		new(big.Int).SetUint64(leftoverGas),
		weiPerGas,
	)
	leftoverMicronibi := evm.WeiToNative(leftoverWei)

	switch leftoverMicronibi.Sign() {
	case -1:
		// Should be impossible since leftoverGas is a uint64. Reaching this case
		// would imply a critical error in the effective gas calculation.
		return errors.Wrapf(evm.ErrInvalidRefund,
			"refunded amount value cannot be negative %s", leftoverMicronibi,
		)
	case 1:
		refundedCoins := sdk.Coins{sdk.NewCoin(evm.EVMBankDenom, sdkmath.NewIntFromBigInt(leftoverMicronibi))}

		// Refund to sender from the fee collector module account. This account
		// manages the collection of gas fees.
		err := k.Bank.SendCoinsFromModuleToAccount(
			ctx,
			authtypes.FeeCollectorName, // sender
			msgFrom.Bytes(),            // recipient
			refundedCoins,
		)
		if err != nil {
			err = errors.Wrapf(errortypes.ErrInsufficientFunds, "fee collector account failed to refund fees: %s", err.Error())
			return errors.Wrapf(err, "failed to refund %d leftover gas (%s)", leftoverGas, refundedCoins.String())
		}
	default:
		// no refund
	}

	return nil
}

// ResetGasMeterAndConsumeGas reset first the gas meter consumed value to zero
// and set it back to the new value 'gasUsed'.
func (k *Keeper) ResetGasMeterAndConsumeGas(ctx sdk.Context, gasUsed uint64) {
	// reset the gas count
	ctx.GasMeter().RefundGas(ctx.GasMeter().GasConsumed(), "reset the gas count")
	ctx.GasMeter().ConsumeGas(gasUsed, "apply evm transaction")
}

// GasToRefund calculates the amount of gas the state machine should refund to
// the sender. It is capped by the refund quotient value. Note that passing a
// jrefundQuotient of 0 will cause problems.
func GasToRefund(availableRefund, gasConsumed, refundQuotient uint64) uint64 {
	// Apply refund counter
	refund := gasConsumed / refundQuotient
	if refund > availableRefund {
		return availableRefund
	}
	return refund
}

// CheckSenderBalance validates that the tx cost value is positive and that the
// sender has enough funds to pay for the fees and value of the transaction.
func CheckSenderBalance(
	balanceWei *big.Int,
	txData evm.TxData,
) error {
	cost := txData.Cost()

	if cost.Sign() < 0 {
		return errors.Wrapf(
			errortypes.ErrInvalidCoins,
			"tx cost (%s) is negative and invalid", cost,
		)
	}

	if balanceWei.Cmp(big.NewInt(0)) < 0 || balanceWei.Cmp(cost) < 0 {
		return errors.Wrapf(
			errortypes.ErrInsufficientFunds,
			"sender balance < tx cost (%s < %s)", balanceWei, cost,
		)
	}
	return nil
}

// DeductTxCostsFromUserBalance deducts the fees from the user balance. Returns
// an error if the specified sender address does not exist or the account balance
// is not sufficient.
func (k *Keeper) DeductTxCostsFromUserBalance(
	ctx sdk.Context,
	fees sdk.Coins,
	from gethcommon.Address,
) error {
	// fetch sender account
	signerAcc, err := authante.GetSignerAcc(ctx, k.accountKeeper, from.Bytes())
	if err != nil {
		return errors.Wrapf(err, "account not found for sender %s", from)
	}

	// deduct the full gas cost from the user balance
	if err := authante.DeductFees(k.Bank, ctx, signerAcc, fees); err != nil {
		return errors.Wrapf(err, "failed to deduct full gas cost %s from the user %s balance", fees, from)
	}

	return nil
}

// VerifyFee is used to return the fee for the given transaction data in
// sdk.Coins. It checks that the gas limit is not reached, the gas limit is
// higher than the intrinsic gas and that the base fee is lower than the gas fee
// cap.
func VerifyFee(
	txData evm.TxData,
	denom string,
	baseFeeMicronibi *big.Int,
	isCheckTx bool,
) (sdk.Coins, error) {
	isContractCreation := txData.GetTo() == nil

	gasLimit := txData.GetGas()

	var accessList gethcore.AccessList
	if txData.GetAccessList() != nil {
		accessList = txData.GetAccessList()
	}

	intrinsicGas, err := core.IntrinsicGas(txData.GetData(), accessList, isContractCreation, true, true)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"failed to retrieve intrinsic gas, contract creation = %t",
			isContractCreation,
		)
	}

	// intrinsic gas verification during CheckTx
	if isCheckTx && gasLimit < intrinsicGas {
		return nil, errors.Wrapf(
			errortypes.ErrOutOfGas,
			"gas limit too low: %d (gas limit) < %d (intrinsic gas)", gasLimit, intrinsicGas,
		)
	}

	if baseFeeMicronibi == nil {
		baseFeeMicronibi = evm.BASE_FEE_MICRONIBI
	}

	// gasFeeCapMicronibi := evm.WeiToNative(txData.GetGasFeeCapWei())
	// if baseFeeMicronibi != nil && gasFeeCapMicronibi.Cmp(baseFeeMicronibi) < 0 {
	// 	baseFeeWei := evm.NativeToWei(baseFeeMicronibi)
	// 	return nil, errors.Wrapf(errortypes.ErrInsufficientFee,
	// 		"the tx gasfeecap is lower than the tx baseFee: %s (gasfeecap), %s (basefee) wei per gas",
	// 		txData.GetGasFeeCapWei(),
	// 		baseFeeWei,
	// 	)
	// }

	baseFeeWei := evm.NativeToWei(baseFeeMicronibi)
	feeAmtMicronibi := evm.WeiToNative(txData.EffectiveFeeWei(baseFeeWei))
	if feeAmtMicronibi.Sign() == 0 {
		// zero fee, no need to deduct
		return sdk.Coins{{Denom: denom, Amount: sdkmath.ZeroInt()}}, nil
	}

	return sdk.Coins{{Denom: denom, Amount: sdkmath.NewIntFromBigInt(feeAmtMicronibi)}}, nil
}
