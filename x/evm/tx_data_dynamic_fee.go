// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethmath "github.com/ethereum/go-ethereum/common/math"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/eth"
)

// BigIntMax returns max(x,y).
func BigIntMax(x, y *big.Int) *big.Int {
	if x == nil && y != nil {
		return y
	} else if x != nil && y == nil {
		return x
	} else if x == nil && y == nil {
		return nil
	}

	if x.Cmp(y) > 0 {
		return x
	}
	return y
}

func NewDynamicFeeTx(tx *gethcore.Transaction) (*DynamicFeeTx, error) {
	txData := &DynamicFeeTx{
		Nonce:    tx.Nonce(),
		Data:     tx.Data(),
		GasLimit: tx.Gas(),
	}

	v, r, s := tx.RawSignatureValues()
	if to := tx.To(); to != nil {
		txData.To = to.Hex()
	}

	if tx.Value() != nil {
		amountInt, err := eth.SafeNewIntFromBigInt(tx.Value())
		if err != nil {
			return nil, err
		}
		txData.Amount = &amountInt
	}

	if tx.GasFeeCap() != nil {
		gasFeeCapInt, err := eth.SafeNewIntFromBigInt(tx.GasFeeCap())
		if err != nil {
			return nil, err
		}
		txData.GasFeeCap = &gasFeeCapInt
	}

	if tx.GasTipCap() != nil {
		gasTipCapInt, err := eth.SafeNewIntFromBigInt(tx.GasTipCap())
		if err != nil {
			return nil, err
		}
		txData.GasTipCap = &gasTipCapInt
	}

	if tx.AccessList() != nil {
		al := tx.AccessList()
		txData.Accesses = NewAccessList(&al)
	}

	txData.SetSignatureValues(tx.ChainId(), v, r, s)
	return txData, nil
}

// TxType returns the tx type
func (tx *DynamicFeeTx) TxType() uint8 {
	return gethcore.DynamicFeeTxType
}

// Copy returns an instance with the same field values
func (tx *DynamicFeeTx) Copy() TxData {
	return &DynamicFeeTx{
		ChainID:   tx.ChainID,
		Nonce:     tx.Nonce,
		GasTipCap: tx.GasTipCap,
		GasFeeCap: tx.GasFeeCap,
		GasLimit:  tx.GasLimit,
		To:        tx.To,
		Amount:    tx.Amount,
		Data:      common.CopyBytes(tx.Data),
		Accesses:  tx.Accesses,
		V:         common.CopyBytes(tx.V),
		R:         common.CopyBytes(tx.R),
		S:         common.CopyBytes(tx.S),
	}
}

// GetChainID returns the chain id field from the DynamicFeeTx
func (tx *DynamicFeeTx) GetChainID() *big.Int {
	if tx.ChainID == nil {
		return nil
	}

	return tx.ChainID.BigInt()
}

// GetAccessList returns the AccessList field.
func (tx *DynamicFeeTx) GetAccessList() gethcore.AccessList {
	if tx.Accesses == nil {
		return nil
	}
	return *tx.Accesses.ToEthAccessList()
}

// GetData returns a copy of the input data bytes.
func (tx *DynamicFeeTx) GetData() []byte {
	return common.CopyBytes(tx.Data)
}

// GetGas returns the gas limit.
func (tx *DynamicFeeTx) GetGas() uint64 {
	return tx.GasLimit
}

// Gas price as wei spent per unit gas.
func (tx *DynamicFeeTx) GetGasPrice() *big.Int {
	return tx.GetGasFeeCapWei()
}

// GetGasTipCapWei returns a cap on the gas tip in units of wei.
//
// Also called "maxPriorityFeePerGas" in Alchemy and Ethers.
// See [Alchemy Docs - maxPriorityFeePerGas vs maxFeePerGas].
// Base fees are determined by the network, not the end user that broadcasts
// the transaction. Adding a tip increases one's "priority" in the block.
//
// The terminology "fee per gas" essentially means "wei per unit gas".
// See [Alchemy Docs - maxPriorityFeePerGas vs maxFeePerGas] for more info.
//
// [Alchemy Docs - maxPriorityFeePerGas vs maxFeePerGas]: https://docs.alchemy.com/docs/maxpriorityfeepergas-vs-maxfeepergas.
func (tx *DynamicFeeTx) GetGasTipCapWei() *big.Int {
	if tx.GasTipCap == nil {
		return nil
	}
	return tx.GasTipCap.BigInt()
}

// GetGasFeeCapWei returns a cap on the gas fees paid in units of wei, where:
//
// feesWithoutCap := effective gas price (wei per gas) * gas units
// gas fee cap -> min(feesWithoutCap, gasFeeCap)
//
// Also called "maxFeePerGas" in Alchemy and Ethers.
// maxFeePerGas := baseFeePerGas + maxPriorityFeePerGas
//
// The terminology "fee per gas" essentially means "wei per unit gas".
// See [Alchemy Docs - maxPriorityFeePerGas vs maxFeePerGas] for more info.
//
// [Alchemy Docs - maxPriorityFeePerGas vs maxFeePerGas]: https://docs.alchemy.com/docs/maxpriorityfeepergas-vs-maxfeepergas.
func (tx *DynamicFeeTx) GetGasFeeCapWei() *big.Int {
	if tx.GasFeeCap == nil {
		return nil
	}
	return tx.GasFeeCap.BigInt()
}

// GetValueWei returns the tx amount.
func (tx *DynamicFeeTx) GetValueWei() *big.Int {
	if tx.Amount == nil {
		return nil
	}

	return tx.Amount.BigInt()
}

// GetNonce returns the account sequence for the transaction.
func (tx *DynamicFeeTx) GetNonce() uint64 { return tx.Nonce }

// GetTo returns the pointer to the recipient address.
func (tx *DynamicFeeTx) GetTo() *common.Address {
	if tx.To == "" {
		return nil
	}
	to := common.HexToAddress(tx.To)
	return &to
}

// AsEthereumData returns an DynamicFeeTx transaction tx from the proto-formatted
// TxData defined on the Cosmos EVM.
func (tx *DynamicFeeTx) AsEthereumData() gethcore.TxData {
	v, r, s := tx.GetRawSignatureValues()
	return &gethcore.DynamicFeeTx{
		ChainID:    tx.GetChainID(),
		Nonce:      tx.GetNonce(),
		GasTipCap:  tx.GetGasTipCapWei(),
		GasFeeCap:  tx.GetGasFeeCapWei(),
		Gas:        tx.GetGas(),
		To:         tx.GetTo(),
		Value:      tx.GetValueWei(),
		Data:       tx.GetData(),
		AccessList: tx.GetAccessList(),
		V:          v,
		R:          r,
		S:          s,
	}
}

// GetRawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *DynamicFeeTx) GetRawSignatureValues() (v, r, s *big.Int) {
	return rawSignatureValues(tx.V, tx.R, tx.S)
}

// SetSignatureValues sets the signature values to the transaction.
func (tx *DynamicFeeTx) SetSignatureValues(chainID, v, r, s *big.Int) {
	if v != nil {
		tx.V = v.Bytes()
	}
	if r != nil {
		tx.R = r.Bytes()
	}
	if s != nil {
		tx.S = s.Bytes()
	}
	if chainID != nil {
		chainIDInt := sdkmath.NewIntFromBigInt(chainID)
		tx.ChainID = &chainIDInt
	}
}

// Validate performs a stateless validation of the tx fields.
func (tx DynamicFeeTx) Validate() error {
	if tx.GasTipCap == nil {
		return errorsmod.Wrap(ErrInvalidGasCap, "gas tip cap cannot nil")
	}

	if tx.GasFeeCap == nil {
		return errorsmod.Wrap(ErrInvalidGasCap, "gas fee cap cannot nil")
	}

	if tx.GasTipCap.IsNegative() {
		return errorsmod.Wrapf(ErrInvalidGasCap, "gas tip cap cannot be negative %s", tx.GasTipCap)
	}

	if tx.GasFeeCap.IsNegative() {
		return errorsmod.Wrapf(ErrInvalidGasCap, "gas fee cap cannot be negative %s", tx.GasFeeCap)
	}

	if !eth.IsValidInt256(tx.GetGasTipCapWei()) {
		return errorsmod.Wrap(ErrInvalidGasCap, "out of bound")
	}

	if !eth.IsValidInt256(tx.GetGasFeeCapWei()) {
		return errorsmod.Wrap(ErrInvalidGasCap, "out of bound")
	}

	if tx.GasFeeCap.LT(*tx.GasTipCap) {
		return errorsmod.Wrapf(
			ErrInvalidGasCap, "max priority fee per gas higher than max fee per gas (%s > %s)",
			tx.GasTipCap, tx.GasFeeCap,
		)
	}

	if !eth.IsValidInt256(tx.Fee()) {
		return errorsmod.Wrap(ErrInvalidGasFee, "out of bound")
	}

	for _, err := range []error{
		ValidateTxDataAmount(&tx),
		ValidateTxDataTo(&tx),
		ValidateTxDataChainID(&tx),
	} {
		if err != nil {
			return err
		}
	}

	return nil
}

// Fee returns gasprice * gaslimit.
func (tx DynamicFeeTx) Fee() *big.Int {
	return priceTimesGas(tx.GetGasFeeCapWei(), tx.GasLimit)
}

// Cost returns amount + gasprice * gaslimit.
func (tx DynamicFeeTx) Cost() *big.Int {
	return cost(tx.Fee(), tx.GetValueWei())
}

// EffectiveGasPriceWeiPerGas returns the effective gas price based on EIP-1559 rules.
// `effectiveGasPrice = min(baseFee + tipCap, feeCap)`
func (tx *DynamicFeeTx) EffectiveGasPriceWeiPerGas(baseFeeWei *big.Int) *big.Int {
	feeWithSpecifiedTip := new(big.Int).Add(tx.GasTipCap.BigInt(), baseFeeWei)

	// Enforce base fee as the minimum [EffectiveGasPriceWei]:
	rawEffectiveGasPrice := gethmath.BigMin(feeWithSpecifiedTip, tx.GasFeeCap.BigInt())
	return BigIntMax(baseFeeWei, rawEffectiveGasPrice)
}

// EffectiveFeeWei returns effective_gasprice * gaslimit.
func (tx DynamicFeeTx) EffectiveFeeWei(baseFeeWei *big.Int) *big.Int {
	return priceTimesGas(tx.EffectiveGasPriceWeiPerGas(baseFeeWei), tx.GasLimit)
}

// EffectiveCostWei returns amount + effective_gasprice * gaslimit.
func (tx DynamicFeeTx) EffectiveCostWei(baseFeeWei *big.Int) *big.Int {
	return cost(tx.EffectiveFeeWei(baseFeeWei), tx.GetValueWei())
}
