// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

// CheckSenderBalance validates that the tx cost value is positive and that the
// sender has enough funds to pay for the fees and value of the transaction.
func CheckSenderBalance(
	balanceWei *uint256.Int,
	txData evm.TxData,
) error {
	cost := txData.Cost()

	if cost.Sign() < 0 {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrInvalidCoins,
			"tx cost (%s) is negative and invalid", cost,
		)
	}

	if balanceWei.ToBig().Cmp(cost) < 0 {
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
func DeductTxCostsFromUserBalance(
	sdb *evmstate.SDB,
	costInWei *uint256.Int,
	from gethcommon.Address,
) error {
	// fetch sender balance
	balWei := sdb.GetBalance(from)

	// deduct the full gas cost from the user balance
	if balWei.Cmp(costInWei) < 0 {
		return fmt.Errorf(`insufficient funds: failed to deduct full gas cost %s: balance="%s", user="%s"`, costInWei, balWei, from)
	}

	sdb.SubBalance(from, costInWei, tracing.BalanceDecreaseGasBuy)
	sdb.AddBalance(evm.FEE_COLLECTOR_ADDR, costInWei, tracing.BalanceDecreaseGasBuy)

	return nil
}
