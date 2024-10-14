// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
)

var (
	_ TxData = &LegacyTx{}
	_ TxData = &AccessListTx{}
	_ TxData = &DynamicFeeTx{}
)

// TxData is the underlying data of a transaction. Its counterpart with private
// fields, "gethcore.TxData" is implemented by DynamicFeeTx, LegacyTx and
// AccessListTx from the same package. Each trnsaction type is implemented here
// for protobuf marshaling.
//
// According to https://github.com/ethereum/go-ethereum/issues/23154:
// TxData exists for the sole purpose of making it easier to construct a
// "gethcore.Transaction" more conviently in Go code. The methods of TxData are
// an internal implementation detail and will never have a stable API.
//
// Because the fields are private in the go-ethereum code, it is impossible to
// provide custom implementations for these methods without creating a new TxData
// data structure. Thus, the current interface exists.
type TxData interface {
	TxType() byte
	Copy() TxData
	GetChainID() *big.Int
	GetAccessList() gethcore.AccessList
	GetData() []byte

	GetNonce() uint64

	// GetGas returns the gas limit in gas units. Note that this is not a "fee"
	// in wei or micronibi or a price.
	GetGas() uint64

	// Gas price as wei spent per unit gas.
	GetGasPrice() *big.Int

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
	GetGasTipCapWei() *big.Int

	// GetGasFeeCapWei returns a cap on the gas fees paid in units of wei, where:
	// feesWithoutCap := effective gas price (wei per gas) * gas units
	// fees -> min(feesWithoutCap, gasFeeCap)
	// Also called "maxFeePerGas" in Alchemy and Ethers.
	//
	// maxFeePerGas := baseFeePerGas + maxPriorityFeePerGas
	//
	// The terminology "fee per gas" essentially means "wei per unit gas".
	// See [Alchemy Docs - maxPriorityFeePerGas vs maxFeePerGas] for more info.
	//
	// [Alchemy Docs - maxPriorityFeePerGas vs maxFeePerGas]: https://docs.alchemy.com/docs/maxpriorityfeepergas-vs-maxfeepergas.
	GetGasFeeCapWei() *big.Int
	EffectiveGasFeeCapWei(baseFeeWei *big.Int) *big.Int

	// GetValueWei: amount of ether (wei units) sent in the transaction.
	GetValueWei() *big.Int

	GetTo() *common.Address
	GetToRaw() string

	GetRawSignatureValues() (v, r, s *big.Int)
	SetSignatureValues(chainID, v, r, s *big.Int)

	AsEthereumData() gethcore.TxData
	Validate() error

	// static fee
	Fee() *big.Int
	// Cost is the gas cost of the transaction in wei
	Cost() *big.Int

	// effective gasPrice/fee/cost according to current base fee
	EffectiveGasPriceWeiPerGas(baseFeeWei *big.Int) *big.Int
	EffectiveFeeWei(baseFeeWei *big.Int) *big.Int
	EffectiveCostWei(baseFeeWei *big.Int) *big.Int
}

// NOTE: All non-protected transactions (i.e. non EIP155 signed) will fail if the
// AllowUnprotectedTxs parameter is disabled.
func NewTxDataFromTx(tx *gethcore.Transaction) (TxData, error) {
	var txData TxData
	var err error
	switch tx.Type() {
	case gethcore.DynamicFeeTxType:
		txData, err = NewDynamicFeeTx(tx)
	case gethcore.AccessListTxType:
		txData, err = newAccessListTx(tx)
	default:
		txData, err = NewLegacyTx(tx)
	}
	if err != nil {
		return nil, err
	}

	return txData, nil
}

// DeriveChainID derives the chain id from the given v parameter.
//
// CONTRACT: v value is either:
//
//   - {0,1} + CHAIN_ID * 2 + 35, if EIP155 is used
//   - {0,1} + 27, otherwise
//
// Ref: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md
func DeriveChainID(v *big.Int) *big.Int {
	if v == nil || v.Sign() < 1 {
		return nil
	}

	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}

		if v < 35 {
			return nil
		}

		// V MUST be of the form {0,1} + CHAIN_ID * 2 + 35
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}

func rawSignatureValues(vBz, rBz, sBz []byte) (v, r, s *big.Int) {
	if len(vBz) > 0 {
		v = new(big.Int).SetBytes(vBz)
	}
	if len(rBz) > 0 {
		r = new(big.Int).SetBytes(rBz)
	}
	if len(sBz) > 0 {
		s = new(big.Int).SetBytes(sBz)
	}
	return v, r, s
}

// Returns the fee in wei corresponding to the given gas price and gas amount.
// Args:
//   - weiPerGas: Wei per unit gas (gas price).
//   - gas: gas units
func priceTimesGas(weiPerGas *big.Int, gas uint64) *big.Int {
	gasLimit := new(big.Int).SetUint64(gas)
	return new(big.Int).Mul(weiPerGas, gasLimit)
}

func cost(fee, value *big.Int) *big.Int {
	if value != nil {
		return new(big.Int).Add(fee, value)
	}
	return fee
}

func (tx *DynamicFeeTx) GetToRaw() string { return tx.To }
func (tx *LegacyTx) GetToRaw() string     { return tx.To }
func (tx *AccessListTx) GetToRaw() string { return tx.To }

func ValidateTxDataAmount(txData TxData) error {
	amount := txData.GetValueWei()
	// Amount can be 0
	if amount != nil && amount.Sign() == -1 {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount cannot be negative %s", amount)
	}
	if !eth.IsValidInt256(amount) {
		return errorsmod.Wrap(ErrInvalidAmount, "out of bound")
	}
	return nil
}

func ValidateTxDataTo(txData TxData) error {
	to := txData.GetToRaw()
	if to != "" {
		if err := eth.ValidateAddress(to); err != nil {
			return errorsmod.Wrap(err, "invalid to address")
		}
	}
	return nil
}

func ValidateTxDataGasPrice(txData TxData) error {
	gasPrice := txData.GetGasPrice()
	if gasPrice == nil {
		return errorsmod.Wrap(ErrInvalidGasPrice, "cannot be nil")
	}
	if !eth.IsValidInt256(gasPrice) {
		return errorsmod.Wrap(ErrInvalidGasPrice, "out of bound")
	}

	if gasPrice.Sign() == -1 {
		return errorsmod.Wrapf(ErrInvalidGasPrice, "gas price cannot be negative %s", gasPrice)
	}
	return nil
}

func ValidateTxDataChainID(txData TxData) error {
	chainID := txData.GetChainID()

	if chainID == nil {
		return errorsmod.Wrap(
			sdkerrors.ErrInvalidChainID,
			"chain ID must be derived from TxData txs",
		)
	}
	return nil
}
