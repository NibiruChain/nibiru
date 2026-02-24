// Copyright (c) 2023-2024 Nibi, Inc.
package evmstate

import (
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/holiman/uint256"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// RefundGas transfers the leftover gas to the sender of the message.
func (k *Keeper) RefundGas(
	sdb *SDB,
	msgFrom gethcommon.Address,
	leftoverGas uint64,
	weiPerGas *big.Int,
) error {
	// Return EVM tokens for remaining gas, exchanged at the original rate.
	leftoverWei := new(big.Int).Mul(
		new(big.Int).SetUint64(leftoverGas),
		weiPerGas,
	)

	switch leftoverWei.Sign() {
	case -1:
		// Should be impossible since leftoverGas is a uint64. Reaching this case
		// would imply a critical error in the effective gas calculation.
		return sdkioerrors.Wrapf(evm.ErrInvalidRefund,
			"refunded amount value cannot be negative %s", leftoverWei,
		)
	case 1:
		wei := uint256.MustFromBig(leftoverWei)

		if balFeeColl := sdb.GetBalance(evm.FEE_COLLECTOR_ADDR); balFeeColl.Cmp(wei) < 0 {
			err := sdkioerrors.Wrapf(
				sdkerrors.ErrInsufficientFunds,
				"fee collector account failed to refund: refund=\"%s wei\" (attonibi), leftover gas=\"%d\"",
				wei, leftoverGas,
			)
			return err
		}

		sdb.SubBalance(evm.FEE_COLLECTOR_ADDR, wei, tracing.BalanceIncreaseGasReturn)
		sdb.AddBalance(msgFrom, wei, tracing.BalanceIncreaseGasReturn)

		// Commit writes this SDB's cached balance changes toward the root context so
		// the refund is persisted. Without it, the refund would remain only in cache.
		sdb.Commit()

		// refundedCoins := sdk.Coins{sdk.NewCoin(evm.EVMBankDenom, sdkmath.NewIntFromBigInt(leftoverMicronibi))}

		// // Refund to sender from the fee collector module account. This account
		// // manages the collection of gas fees.
		// err := k.Bank.SendCoinsFromModuleToAccount(
		// 	sdb.Ctx(),
		// 	authtypes.FeeCollectorName, // sender
		// 	msgFrom.Bytes(),            // recipient
		// 	refundedCoins,
		// )
		// if err != nil {
		// 	err = sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "fee collector account failed to refund fees: %s", err.Error())
		// 	return sdkioerrors.Wrapf(err, "failed to refund %d leftover gas (%s)", leftoverGas, refundedCoins.String())
		// }
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

// VerifyFee is used to return the fee, or token payment, for the given
// transaction data in [sdk.Coin]s. It checks that the gas limit and uses the
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
) (effFeeWei *uint256.Int, err error) {
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
	feeAmtWei := txData.EffectiveFeeWei(baseFeeWei)
	if feeAmtWei.Sign() <= 0 {
		// zero fee, no need to deduct
		return uint256.NewInt(0), nil
	}
	return uint256.MustFromBig(feeAmtWei), nil
}

func Rules(ctx sdk.Context) gethparams.Rules {
	chainConfig := evm.EthereumConfig(appconst.GetEthChainID(ctx.ChainID()))
	return chainConfig.Rules(
		big.NewInt(ctx.BlockHeight()),
		false, // isMerge
		evm.ParseBlockTimeUnixU64(ctx),
	)
}
