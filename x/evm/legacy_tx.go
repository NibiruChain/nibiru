// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"math/big"

	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	errorsmod "cosmossdk.io/errors"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
)

func NewLegacyTx(tx *gethcore.Transaction) (*LegacyTx, error) {
	txData := &LegacyTx{
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

	if tx.GasPrice() != nil {
		gasPriceInt, err := eth.SafeNewIntFromBigInt(tx.GasPrice())
		if err != nil {
			return nil, err
		}
		txData.GasPrice = &gasPriceInt
	}

	txData.SetSignatureValues(tx.ChainId(), v, r, s)
	return txData, nil
}

// TxType returns the tx type
func (tx *LegacyTx) TxType() uint8 {
	return gethcore.LegacyTxType
}

// Copy returns an instance with the same field values
func (tx *LegacyTx) Copy() TxData {
	return &LegacyTx{
		Nonce:    tx.Nonce,
		GasPrice: tx.GasPrice,
		GasLimit: tx.GasLimit,
		To:       tx.To,
		Amount:   tx.Amount,
		Data:     common.CopyBytes(tx.Data),
		V:        common.CopyBytes(tx.V),
		R:        common.CopyBytes(tx.R),
		S:        common.CopyBytes(tx.S),
	}
}

// GetChainID returns the chain id field from the derived signature values
func (tx *LegacyTx) GetChainID() *big.Int {
	v, _, _ := tx.GetRawSignatureValues()
	return DeriveChainID(v)
}

// GetAccessList returns nil
func (tx *LegacyTx) GetAccessList() gethcore.AccessList {
	return nil
}

// GetData returns a copy of the input data bytes.
func (tx *LegacyTx) GetData() []byte {
	return common.CopyBytes(tx.Data)
}

// GetGas returns the gas limit.
func (tx *LegacyTx) GetGas() uint64 {
	return tx.GasLimit
}

// GetGasPrice is equivalent to wei per unit gas.
func (tx *LegacyTx) GetGasPrice() *big.Int {
	if tx.GasPrice == nil {
		return nil
	}
	return tx.GasPrice.BigInt()
}

// GetGasTipCapWei returns the gas price field.
func (tx *LegacyTx) GetGasTipCapWei() *big.Int {
	return tx.GetGasPrice()
}

// GetGasFeeCapWei returns the gas price field.
func (tx *LegacyTx) GetGasFeeCapWei() *big.Int {
	return tx.GetGasPrice()
}

// GetValueWei returns the tx amount.
func (tx *LegacyTx) GetValueWei() *big.Int {
	if tx.Amount == nil {
		return nil
	}
	return tx.Amount.BigInt()
}

// GetNonce returns the account sequence for the transaction.
func (tx *LegacyTx) GetNonce() uint64 { return tx.Nonce }

// GetTo returns the pointer to the recipient address.
func (tx *LegacyTx) GetTo() *common.Address {
	if tx.To == "" {
		return nil
	}
	to := common.HexToAddress(tx.To)
	return &to
}

// AsEthereumData returns an LegacyTx transaction tx from the proto-formatted
// TxData defined on the Cosmos EVM.
func (tx *LegacyTx) AsEthereumData() gethcore.TxData {
	v, r, s := tx.GetRawSignatureValues()
	return &gethcore.LegacyTx{
		Nonce:    tx.GetNonce(),
		GasPrice: tx.GetGasPrice(),
		Gas:      tx.GetGas(),
		To:       tx.GetTo(),
		Value:    tx.GetValueWei(),
		Data:     tx.GetData(),
		V:        v,
		R:        r,
		S:        s,
	}
}

// GetRawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *LegacyTx) GetRawSignatureValues() (v, r, s *big.Int) {
	return rawSignatureValues(tx.V, tx.R, tx.S)
}

// SetSignatureValues sets the signature values to the transaction.
func (tx *LegacyTx) SetSignatureValues(_, v, r, s *big.Int) {
	if v != nil {
		tx.V = v.Bytes()
	}
	if r != nil {
		tx.R = r.Bytes()
	}
	if s != nil {
		tx.S = s.Bytes()
	}
}

// Validate performs a stateless validation of the tx fields.
func (tx LegacyTx) Validate() error {
	gasPrice := tx.GetGasPrice()
	if gasPrice == nil {
		return errorsmod.Wrap(ErrInvalidGasPrice, "gas price cannot be nil")
	}

	if gasPrice.Sign() == -1 {
		return errorsmod.Wrapf(ErrInvalidGasPrice, "gas price cannot be negative %s", gasPrice)
	}
	if !eth.IsValidInt256(gasPrice) {
		return errorsmod.Wrap(ErrInvalidGasPrice, "out of bound")
	}
	if !eth.IsValidInt256(tx.Fee()) {
		return errorsmod.Wrap(ErrInvalidGasFee, "out of bound")
	}

	amount := tx.GetValueWei()
	// Amount can be 0
	if amount != nil && amount.Sign() == -1 {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount cannot be negative %s", amount)
	}
	if !eth.IsValidInt256(amount) {
		return errorsmod.Wrap(ErrInvalidAmount, "out of bound")
	}

	if tx.To != "" {
		if err := eth.ValidateAddress(tx.To); err != nil {
			return errorsmod.Wrap(err, "invalid to address")
		}
	}

	chainID := tx.GetChainID()

	if chainID == nil {
		return errorsmod.Wrap(
			errortypes.ErrInvalidChainID,
			"chain ID must be derived from LegacyTx txs",
		)
	}

	return nil
}

// Fee returns gasprice * gaslimit.
func (tx LegacyTx) Fee() *big.Int {
	return fee(tx.GetGasPrice(), tx.GetGas())
}

// Cost returns amount + gasprice * gaslimit.
func (tx LegacyTx) Cost() *big.Int {
	return cost(tx.Fee(), tx.GetValueWei())
}

// EffectiveGasPriceWei is the same as GasPrice for LegacyTx
func (tx LegacyTx) EffectiveGasPriceWei(_ *big.Int) *big.Int {
	return tx.GetGasPrice()
}

// EffectiveFeeWei is the same as Fee for LegacyTx
func (tx LegacyTx) EffectiveFeeWei(_ *big.Int) *big.Int {
	return tx.Fee()
}

// EffectiveCost is the same as Cost for LegacyTx
func (tx LegacyTx) EffectiveCost(_ *big.Int) *big.Int {
	return tx.Cost()
}
