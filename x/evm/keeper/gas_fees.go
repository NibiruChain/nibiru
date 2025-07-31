// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

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
		return sdkioerrors.Wrapf(evm.ErrInvalidRefund,
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
			err = sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "fee collector account failed to refund fees: %s", err.Error())
			return sdkioerrors.Wrapf(err, "failed to refund %d leftover gas (%s)", leftoverGas, refundedCoins.String())
		}
	default:
		// no refund
	}

	return nil
}

// gasToRefund calculates the amount of gas the state machine should refund to
// the sender.
// EIP-3529: refunds are capped to gasUsed / 5
func gasToRefund(availableRefundAmount, gasUsed uint64) uint64 {
	refundAmount := gasUsed / gethparams.RefundQuotientEIP3529
	if refundAmount > availableRefundAmount {
		// Apply refundAmount counter
		return availableRefundAmount
	}
	return refundAmount
}

// CheckSenderBalance validates that the tx cost value is positive and that the
// sender has enough funds to pay for the fees and value of the transaction.
func CheckSenderBalance(
	balanceWei *big.Int,
	txData evm.TxData,
) error {
	cost := txData.Cost()

	if cost.Sign() < 0 {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrInvalidCoins,
			"tx cost (%s) is negative and invalid", cost,
		)
	}

	if balanceWei.Cmp(big.NewInt(0)) < 0 || balanceWei.Cmp(cost) < 0 {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrInsufficientFunds,
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
		return sdkioerrors.Wrapf(err, "account not found for sender %s", from)
	}

	// deduct the full gas cost from the user balance
	if err := authante.DeductFees(k.Bank, ctx, signerAcc, fees); err != nil {
		return sdkioerrors.Wrapf(err, "failed to deduct full gas cost %s from the user %s balance", fees, from)
	}

	return nil
}

// VerifyFee is used to return the fee, or token payment, for the given
// transaction data in [sdk.Coin]s. It checks that the the gas limit and uses the
// "effective fee" from the [evm.TxData].
//
//   - For [evm.DynamicFeeTx], the effective gas price is the minimum of
//     (baseFee + tipCap) and the gas fee cap (feeCap).
//   - For [evm.LegacyTx] and [evm.AccessListTx], the effective gas price is the
//     max of the gas price and baseFee.
//
// Transactions where the baseFee exceeds the feeCap are priced out
// under EIP-1559 and will not pass validation.
//
// Args:
//   - txData: Tx data related to gas, effectie gas, nonce, and chain ID
//     implemented by every Ethereum tx type.
//   - baseFeeMicronibi:EIP1559 base fee in units of micronibi ("unibi").
//   - isCheckTx: Comes from `[sdk.Context].isCheckTx()`
func VerifyFee(
	txData evm.TxData,
	baseFeeMicronibi *big.Int,
	ctx sdk.Context,
) (sdk.Coins, error) {
	var (
		isContractCreation = txData.GetTo() == nil
		isCheckTx          = ctx.IsCheckTx()
		rules              = Rules(ctx)
	)

	gasLimit := txData.GetGas()

	var accessList gethcore.AccessList
	if txData.GetAccessList() != nil {
		accessList = txData.GetAccessList()
	}

	intrinsicGas, err := core.IntrinsicGas(
		txData.GetData(),
		accessList,
		isContractCreation,
		rules.IsHomestead,
		rules.IsIstanbul, // isEIP2028 === IsInstanbul
		rules.IsShanghai, // isEIP3860 === isShanghai
	)
	if err != nil {
		return nil, sdkioerrors.Wrapf(
			err,
			"failed to retrieve intrinsic gas, contract creation = %t",
			isContractCreation,
		)
	}

	// intrinsic gas verification during CheckTx
	if isCheckTx && gasLimit < intrinsicGas {
		return nil, sdkioerrors.Wrapf(
			sdkerrors.ErrOutOfGas,
			"gas limit too low: %d (gas limit) < %d (intrinsic gas)", gasLimit, intrinsicGas,
		)
	}

	if baseFeeMicronibi == nil {
		baseFeeMicronibi = evm.BASE_FEE_MICRONIBI
	}

	baseFeeWei := evm.NativeToWei(baseFeeMicronibi)
	feeAmtMicronibi := evm.WeiToNative(txData.EffectiveFeeWei(baseFeeWei))
	bankDenom := evm.EVMBankDenom
	if feeAmtMicronibi.Sign() == 0 {
		// zero fee, no need to deduct
		return sdk.Coins{{Denom: bankDenom, Amount: sdkmath.ZeroInt()}}, nil
	}

	return sdk.Coins{{Denom: bankDenom, Amount: sdkmath.NewIntFromBigInt(feeAmtMicronibi)}}, nil
}

func Rules(ctx sdk.Context) gethparams.Rules {
	chainConfig := evm.EthereumConfig(appconst.GetEthChainID(ctx.ChainID()))
	return chainConfig.Rules(
		big.NewInt(ctx.BlockHeight()),
		false, // isMerge
		evm.ParseBlockTimeUnixU64(ctx),
	)
}
