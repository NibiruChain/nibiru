// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EvmTxArgs encapsulates all possible params to create all EVM txs types.
// This includes LegacyTx, DynamicFeeTx and AccessListTx
type EvmTxArgs struct {
	Nonce     uint64
	GasLimit  uint64
	Input     []byte
	GasFeeCap *big.Int
	GasPrice  *big.Int
	ChainID   *big.Int
	Amount    *big.Int
	GasTipCap *big.Int
	To        *common.Address
	Accesses  *gethcore.AccessList
}

// DefaultPriorityReduction is the default amount of price values required for 1 unit of priority.
// Because priority is `int64` while price is `big.Int`, it's necessary to scale down the range to keep it more pratical.
// The default value is the same as the `sdk.DefaultPowerReduction`.
var DefaultPriorityReduction = sdk.DefaultPowerReduction

// GetTxPriority returns the priority of a given Ethereum tx. It relies on the
// priority reduction global variable to calculate the tx priority given the tx
// tip price:
//
//	tx_priority = tip_price / priority_reduction
func GetTxPriority(txData TxData, baseFee *big.Int) (priority int64) {
	// calculate priority based on effective gas price
	tipPrice := txData.EffectiveGasPriceWeiPerGas(baseFee)

	// Return the min of the max possible priorty and the derived priority
	priority = math.MaxInt64
	derivedPriority := new(big.Int).Quo(tipPrice, DefaultPriorityReduction.BigInt())

	// Overflow safety check
	var priorityBigI64 int64
	if derivedPriority.IsInt64() {
		priorityBigI64 = derivedPriority.Int64()
	} else {
		priorityBigI64 = priority
	}
	return min(priority, priorityBigI64)
}

// Failed returns if the contract execution failed in vm errors
func (m *MsgEthereumTxResponse) Failed() bool {
	return len(m.VmError) > 0
}

// Return is a helper function to help caller distinguish between revert reason
// and function return. Return returns the data after execution if no error occurs.
func (m *MsgEthereumTxResponse) Return() []byte {
	if m.Failed() {
		return nil
	}
	return common.CopyBytes(m.Ret)
}

// Revert returns the concrete revert reason if the execution is aborted by `REVERT`
// opcode. Note the reason can be nil if no data supplied with revert opcode.
func (m *MsgEthereumTxResponse) Revert() []byte {
	if m.VmError != vm.ErrExecutionReverted.Error() {
		return nil
	}
	return common.CopyBytes(m.Ret)
}
