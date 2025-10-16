// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

var _ AnteStep = AnteStepValidateBasic

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

	// Tx gas limit must be at least enough for the intrinsic gas cost.
	gasLimit := msgEthTx.GetGas()

	// Compute actual intrinsic gas (same as in ApplyEvmMsg)
	rules := k.GetEVMConfig(sdb.Ctx()).ChainConfig.Rules(
		big.NewInt(sdb.Ctx().BlockHeight()),
		false,
		evm.ParseBlockTimeUnixU64(sdb.Ctx()),
	)
	contractCreation := txData.GetTo() == nil
	intrinsicGasCost, err := core.IntrinsicGas(
		txData.GetData(),
		txData.GetAccessList(),
		contractCreation,
		rules.IsHomestead,
		rules.IsIstanbul,
		rules.IsShanghai,
	)
	if err != nil {
		return fmt.Errorf("failed to compute intrinsic gas: %w", err)
	}

	// Validate against actual intrinsic gas
	if gasLimit < intrinsicGasCost {
		return fmt.Errorf(
			"%s: provided msg.Gas (%d) for the tx gas limit is less than intrinsic gas cost (%d).",
			vm.ErrOutOfGas, gasLimit, intrinsicGasCost,
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
