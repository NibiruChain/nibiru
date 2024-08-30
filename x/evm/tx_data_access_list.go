// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
)

// AccessList is an EIP-2930 access list that represents the slice of
// the protobuf AccessTuples.
type AccessList []AccessTuple

// NewAccessList creates a new protobuf-compatible AccessList from an ethereum
// core AccessList type
func NewAccessList(ethAccessList *gethcore.AccessList) AccessList {
	if ethAccessList == nil {
		return nil
	}

	al := AccessList{}
	for _, tuple := range *ethAccessList {
		storageKeys := make([]string, len(tuple.StorageKeys))

		for i := range tuple.StorageKeys {
			storageKeys[i] = tuple.StorageKeys[i].String()
		}

		al = append(al, AccessTuple{
			Address:     tuple.Address.String(),
			StorageKeys: storageKeys,
		})
	}

	return al
}

// ToEthAccessList is a utility function to convert the protobuf compatible
// AccessList to eth core AccessList from go-ethereum
func (al AccessList) ToEthAccessList() *gethcore.AccessList {
	var ethAccessList gethcore.AccessList

	for _, tuple := range al {
		storageKeys := make([]common.Hash, len(tuple.StorageKeys))

		for i := range tuple.StorageKeys {
			storageKeys[i] = common.HexToHash(tuple.StorageKeys[i])
		}

		ethAccessList = append(ethAccessList, gethcore.AccessTuple{
			Address:     common.HexToAddress(tuple.Address),
			StorageKeys: storageKeys,
		})
	}

	return &ethAccessList
}

// AccessListTx

func newAccessListTx(tx *gethcore.Transaction) (*AccessListTx, error) {
	txData := &AccessListTx{
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

	if tx.AccessList() != nil {
		al := tx.AccessList()
		txData.Accesses = NewAccessList(&al)
	}

	txData.SetSignatureValues(tx.ChainId(), v, r, s)
	return txData, nil
}

// TxType returns the tx type
func (tx *AccessListTx) TxType() uint8 {
	return gethcore.AccessListTxType
}

// Copy returns an instance with the same field values
func (tx *AccessListTx) Copy() TxData {
	return &AccessListTx{
		ChainID:  tx.ChainID,
		Nonce:    tx.Nonce,
		GasPrice: tx.GasPrice,
		GasLimit: tx.GasLimit,
		To:       tx.To,
		Amount:   tx.Amount,
		Data:     common.CopyBytes(tx.Data),
		Accesses: tx.Accesses,
		V:        common.CopyBytes(tx.V),
		R:        common.CopyBytes(tx.R),
		S:        common.CopyBytes(tx.S),
	}
}

// GetChainID returns the chain id field from the AccessListTx
func (tx *AccessListTx) GetChainID() *big.Int {
	if tx.ChainID == nil {
		return nil
	}

	return tx.ChainID.BigInt()
}

// GetAccessList returns the AccessList field.
func (tx *AccessListTx) GetAccessList() gethcore.AccessList {
	if tx.Accesses == nil {
		return nil
	}
	return *tx.Accesses.ToEthAccessList()
}

// GetData returns a copy of the input data bytes.
func (tx *AccessListTx) GetData() []byte {
	return common.CopyBytes(tx.Data)
}

// GetGas returns the gas limit.
func (tx *AccessListTx) GetGas() uint64 {
	return tx.GasLimit
}

// Gas price as wei spent per unit gas.
func (tx *AccessListTx) GetGasPrice() *big.Int {
	if tx.GasPrice == nil {
		return nil
	}
	return tx.GasPrice.BigInt()
}

// GetGasTipCapWei returns a cap on the gas tip in units of wei.
// For an [AccessListTx], this is taken to be the gas price.
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
func (tx *AccessListTx) GetGasTipCapWei() *big.Int {
	return tx.GetGasPrice()
}

// GetGasFeeCapWei returns a cap on the gas fees paid in units of wei:
// For an [AccessListTx], this is taken to be the gas price.
//
// The terminology "fee per gas" essentially means "wei per unit gas".
// See [Alchemy Docs - maxPriorityFeePerGas vs maxFeePerGas] for more info.
//
// [Alchemy Docs - maxPriorityFeePerGas vs maxFeePerGas]: https://docs.alchemy.com/docs/maxpriorityfeepergas-vs-maxfeepergas.
func (tx *AccessListTx) GetGasFeeCapWei() *big.Int {
	return tx.GetGasPrice()
}

// GetValueWei returns the tx amount.
func (tx *AccessListTx) GetValueWei() *big.Int {
	if tx.Amount == nil {
		return nil
	}

	return tx.Amount.BigInt()
}

// GetNonce returns the account sequence for the transaction.
func (tx *AccessListTx) GetNonce() uint64 { return tx.Nonce }

// GetTo returns the pointer to the recipient address.
func (tx *AccessListTx) GetTo() *common.Address {
	if tx.To == "" {
		return nil
	}
	to := common.HexToAddress(tx.To)
	return &to
}

// AsEthereumData returns an AccessListTx transaction tx from the proto-formatted
// TxData defined on the Cosmos EVM.
func (tx *AccessListTx) AsEthereumData() gethcore.TxData {
	v, r, s := tx.GetRawSignatureValues()
	return &gethcore.AccessListTx{
		ChainID:    tx.GetChainID(),
		Nonce:      tx.GetNonce(),
		GasPrice:   tx.GetGasPrice(),
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
func (tx *AccessListTx) GetRawSignatureValues() (v, r, s *big.Int) {
	return rawSignatureValues(tx.V, tx.R, tx.S)
}

// SetSignatureValues sets the signature values to the transaction.
func (tx *AccessListTx) SetSignatureValues(chainID, v, r, s *big.Int) {
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
func (tx AccessListTx) Validate() error {
	for _, err := range []error{
		ValidateTxDataAmount(&tx),
		ValidateTxDataTo(&tx),
		ValidateTxDataGasPrice(&tx),
	} {
		if err != nil {
			return err
		}
	}

	if !eth.IsValidInt256(tx.Fee()) {
		return errorsmod.Wrap(ErrInvalidGasFee, "out of bound")
	}

	chainID := tx.GetChainID()

	if chainID == nil {
		return errorsmod.Wrap(
			errortypes.ErrInvalidChainID,
			"chain ID must be present on AccessList txs",
		)
	}

	return nil
}

// Fee returns gasprice * gaslimit.
func (tx AccessListTx) Fee() *big.Int {
	return priceTimesGas(tx.GetGasPrice(), tx.GetGas())
}

// Cost returns amount + gasprice * gaslimit.
func (tx AccessListTx) Cost() *big.Int {
	return cost(tx.Fee(), tx.GetValueWei())
}

// EffectiveGasPriceWei is the same as GasPrice for AccessListTx
func (tx AccessListTx) EffectiveGasPriceWei(baseFeeWei *big.Int) *big.Int {
	return BigIntMax(tx.GetGasPrice(), baseFeeWei)
}

// EffectiveFeeWei is the same as Fee for AccessListTx
func (tx AccessListTx) EffectiveFeeWei(baseFeeWei *big.Int) *big.Int {
	return priceTimesGas(tx.EffectiveGasPriceWei(baseFeeWei), tx.GetGas())
}

// EffectiveCost is the same as Cost for AccessListTx
func (tx AccessListTx) EffectiveCost(baseFeeWei *big.Int) *big.Int {
	txFee := tx.EffectiveFeeWei(baseFeeWei)
	return cost(txFee, tx.GetValueWei())
}
