// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	// gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

var _ EvmAnteStep = AnteStepValidateBasic

// AnteStepValidateBasic performs basic validation of Ethereum transactions
// within the EVM state transition. This handler validates transaction structure,
// fee calculations, and ensures proper Ethereum transaction format.
//
// This handler will fail if:
//   - the transaction is invalid (basic validation fails)
//   - the transaction structure doesn't match Ethereum format requirements
//   - fee calculations are incorrect
//   - transaction data cannot be unpacked
//   - dynamic fee transactions are used when base fee is not available
//
// The handler validates that the transaction follows the proper Ethereum
// transaction format and that all required fields are correctly set.
func AnteStepValidateBasic(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	if sdb.Ctx().IsReCheckTx() {
		// Skip validation during ReCheckTx to avoid redundant checks
		// We can do this because "ValidateBasic" checks are not stateful,
		// meaning this passed the [EvmAnteStep] during CheckTx.
		return nil
	}

	// Validate the Ethereum transaction message
	if err := msgEthTx.ValidateBasic(); err != nil {
		return fmt.Errorf("tx basic validation failed: %w", err)
	}

	// Unpack and validate transaction data
	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to unpack MsgEthereumTx Data")
	}

	// Validate gas limit is reasonable
	gasLimit := msgEthTx.GetGas()
	if gasLimit < gethparams.TxGas {
		return fmt.Errorf(
			"gas limit must exceed the lowest possible intrinsic gas cost of %d, got gas limit %d.", gethparams.TxGas, gasLimit,
		)
	}

	// Validate effective fee calculation
	baseFeeWei := k.BaseFeeWeiPerGas(sdb.Ctx())
	effectiveFeeWei := txData.EffectiveFeeWei(baseFeeWei)
	if effectiveFeeWei.Sign() < 0 {
		return sdkioerrors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"effective fee cannot be negative",
		)
	}

	return nil
}
