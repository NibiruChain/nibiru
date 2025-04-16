// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/x/common/nmath"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	geth "github.com/ethereum/go-ethereum/core/types"
)

// JsonTxArgs represents the arguments to construct a new transaction
// or a message call using JSON-RPC.
// Duplicate struct definition since geth struct is in internal package
// Ref: https://github.com/ethereum/go-ethereum/blob/release/1.10.4/internal/ethapi/transaction_args.go#L36
type JsonTxArgs struct {
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`

	// We accept "data" and "input" for backwards-compatibility reasons.
	// "input" is the newer name and should be preferred by clients.
	// Issue detail: https://github.com/ethereum/go-ethereum/issues/15628
	Data *hexutil.Bytes `json:"data"`
	// Both "data" and "input" are accepted for backwards-compatibility reasons.
	// "input" is the newer name and should be preferred by clients.
	// Issue detail: https://github.com/ethereum/go-ethereum/issues/15628
	Input *hexutil.Bytes `json:"input"`

	// Introduced by AccessListTxType transaction.
	AccessList *geth.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big     `json:"chainId,omitempty"`
}

// String return the struct in a string format
func (args *JsonTxArgs) String() string {
	// Todo: There is currently a bug with hexutil.Big when the value its nil, printing would trigger an exception
	return fmt.Sprintf("TransactionArgs{From:%v, To:%v, Gas:%v,"+
		" Nonce:%v, Data:%v, Input:%v, AccessList:%v}",
		args.From,
		args.To,
		args.Gas,
		args.Nonce,
		args.Data,
		args.Input,
		args.AccessList)
}

// ToMsgEthTx converts the arguments to an ethereum transaction.
// This assumes that setTxDefaults has been called.
func (args *JsonTxArgs) ToMsgEthTx() *MsgEthereumTx {
	var (
		chainID, value, gasPrice, maxFeePerGas, maxPriorityFeePerGas sdkmath.Int
		gas, nonce                                                   uint64
		from, to                                                     string
	)

	// Set sender address or use zero address if none specified.
	if args.ChainID != nil {
		chainID = sdkmath.NewIntFromBigInt(args.ChainID.ToInt())
	}

	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	}

	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}

	if args.GasPrice != nil {
		gasPrice = sdkmath.NewIntFromBigInt(args.GasPrice.ToInt())
	}

	if args.MaxFeePerGas != nil {
		maxFeePerGas = sdkmath.NewIntFromBigInt(args.MaxFeePerGas.ToInt())
	}

	if args.MaxPriorityFeePerGas != nil {
		maxPriorityFeePerGas = sdkmath.NewIntFromBigInt(args.MaxPriorityFeePerGas.ToInt())
	}

	if args.Value != nil {
		value = sdkmath.NewIntFromBigInt(args.Value.ToInt())
	}

	if args.To != nil {
		to = args.To.Hex()
	}

	var data TxData
	switch {
	case args.MaxFeePerGas != nil:
		al := AccessList{}
		if args.AccessList != nil {
			al = NewAccessList(args.AccessList)
		}

		data = &DynamicFeeTx{
			To:        to,
			ChainID:   &chainID,
			Nonce:     nonce,
			GasLimit:  gas,
			GasFeeCap: &maxFeePerGas,
			GasTipCap: &maxPriorityFeePerGas,
			Amount:    &value,
			Data:      args.GetData(),
			Accesses:  al,
		}
	case args.AccessList != nil:
		data = &AccessListTx{
			To:       to,
			ChainID:  &chainID,
			Nonce:    nonce,
			GasLimit: gas,
			GasPrice: &gasPrice,
			Amount:   &value,
			Data:     args.GetData(),
			Accesses: NewAccessList(args.AccessList),
		}
	default:
		data = &LegacyTx{
			To:       to,
			Nonce:    nonce,
			GasLimit: gas,
			GasPrice: &gasPrice,
			Amount:   &value,
			Data:     args.GetData(),
		}
	}

	anyData, err := PackTxData(data)
	if err != nil {
		return nil
	}

	if args.From != nil {
		from = args.From.Hex()
	}

	msg := MsgEthereumTx{
		Data: anyData,
		From: from,
	}
	msg.Hash = msg.AsTransaction().Hash().Hex()
	return &msg
}

// ToMessage converts the arguments to the Message type used by the core evm.
// This assumes that setTxDefaults has been called.
func (args *JsonTxArgs) ToMessage(globalGasCap uint64, baseFeeWei *big.Int) (core.Message, error) {
	// Reject invalid combinations of pre- and post-1559 fee styles
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return core.Message{}, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}

	// Set sender address or use zero address if none specified.
	addr := args.GetFrom()

	// Set default gas & gas price if none were set
	gas := globalGasCap
	if gas == 0 {
		gas = uint64(math.MaxUint64 / 2)
	}
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if globalGasCap != 0 && globalGasCap < gas {
		gas = globalGasCap
	}
	var (
		gasPrice  *big.Int
		gasFeeCap *big.Int
		gasTipCap *big.Int
	)
	if baseFeeWei == nil {
		// If there's no basefee, then it must be a non-1559 execution
		gasPrice = new(big.Int)
		if args.GasPrice != nil {
			gasPrice = args.GasPrice.ToInt()
		}
		gasFeeCap, gasTipCap = gasPrice, gasPrice
	} else {
		// A basefee is provided, necessitating 1559-type execution
		if args.GasPrice != nil {
			// User specified the legacy gas field, convert to 1559 gas typing
			gasPrice = args.GasPrice.ToInt()
			gasFeeCap, gasTipCap = gasPrice, gasPrice
		} else {
			// User specified 1559 gas feilds (or none), use those
			gasFeeCap = new(big.Int)
			if args.MaxFeePerGas != nil {
				gasFeeCap = args.MaxFeePerGas.ToInt()
			}
			gasTipCap = new(big.Int)
			if args.MaxPriorityFeePerGas != nil {
				gasTipCap = args.MaxPriorityFeePerGas.ToInt()
			}
			// Backfill the legacy gasPrice for EVM execution, unless we're all zeroes
			gasPrice = new(big.Int)
			if gasFeeCap.BitLen() > 0 || gasTipCap.BitLen() > 0 {
				gasPrice = nmath.BigMin(new(big.Int).Add(gasTipCap, baseFeeWei), gasFeeCap)
			}
		}
	}
	value := new(big.Int)
	if args.Value != nil {
		value = args.Value.ToInt()
	}
	data := args.GetData()
	var accessList geth.AccessList
	if args.AccessList != nil {
		accessList = *args.AccessList
	}

	nonce := uint64(0)
	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	}

	evmMsg := core.Message{
		To:               args.To,
		From:             addr,
		Nonce:            nonce,
		Value:            value, // amount
		GasLimit:         gas,
		GasPrice:         gasPrice,
		GasFeeCap:        gasFeeCap,
		GasTipCap:        gasTipCap,
		Data:             data,
		AccessList:       accessList,
		SkipNonceChecks:  true,
		SkipFromEOACheck: true,
	}

	return evmMsg, nil
}

// GetFrom retrieves the transaction sender address.
func (args *JsonTxArgs) GetFrom() common.Address {
	if args.From == nil {
		return common.Address{}
	}
	return *args.From
}

// GetData retrieves the transaction calldata. Input field is preferred.
func (args *JsonTxArgs) GetData() []byte {
	if args.Input != nil {
		return *args.Input
	}
	if args.Data != nil {
		return *args.Data
	}
	return nil
}
