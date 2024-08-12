// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"math/big"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"

	"github.com/NibiruChain/nibiru/x/evm"
)

// RefundGas transfers the leftover gas to the sender of the message, caped to half of the total gas
// consumed in the transaction. Additionally, the function sets the total gas consumed to the value
// returned by the EVM execution, thus ignoring the previous intrinsic gas consumed during in the
// AnteHandler.
func (k *Keeper) RefundGas(ctx sdk.Context, msg core.Message, leftoverGas uint64, denom string) error {
	// Return EVM tokens for remaining gas, exchanged at the original rate.
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(leftoverGas), msg.GasPrice())

	switch remaining.Sign() {
	case -1:
		// negative refund errors
		return errors.Wrapf(evm.ErrInvalidRefund, "refunded amount value cannot be negative %d", remaining.Int64())
	case 1:
		// positive amount refund
		refundedCoins := sdk.Coins{sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(remaining))}

		// refund to sender from the fee collector module account, which is the escrow account in charge of collecting tx fees

		err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.FeeCollectorName, msg.From().Bytes(), refundedCoins)
		if err != nil {
			err = errors.Wrapf(errortypes.ErrInsufficientFunds, "fee collector account failed to refund fees: %s", err.Error())
			return errors.Wrapf(err, "failed to refund %d leftover gas (%s)", leftoverGas, refundedCoins.String())
		}
	default:
		// no refund, consume gas and update the tx gas meter
	}

	return nil
}

// ResetGasMeterAndConsumeGas reset first the gas meter consumed value to zero and set it back to the new value
// 'gasUsed'
func (k *Keeper) ResetGasMeterAndConsumeGas(ctx sdk.Context, gasUsed uint64) {
	// reset the gas count
	ctx.GasMeter().RefundGas(ctx.GasMeter().GasConsumed(), "reset the gas count")
	ctx.GasMeter().ConsumeGas(gasUsed, "apply evm transaction")
}

// gasToRefund calculates the amount of gas the state machine should refund to the sender. It is
// capped by the refund quotient value.
// Note: do not pass 0 to refundQuotient
func gasToRefund(availableRefund, gasConsumed, refundQuotient uint64) uint64 {
	// Apply refund counter
	refund := gasConsumed / refundQuotient
	if refund > availableRefund {
		return availableRefund
	}
	return refund
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
	if err := authante.DeductFees(k.bankKeeper, ctx, signerAcc, fees); err != nil {
		return errors.Wrapf(err, "failed to deduct full gas cost %s from the user %s balance", fees, from)
	}

	return nil
}
