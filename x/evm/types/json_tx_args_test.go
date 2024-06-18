// Copyright (c) 2023-2024 Nibi, Inc.
package types_test

import (
	"fmt"
	"math/big"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcoretypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/x/evm/types"
)

func (suite *TxDataTestSuite) TestTxArgsString() {
	testCases := []struct {
		name           string
		txArgs         types.JsonTxArgs
		expectedString string
	}{
		{
			"empty tx args",
			types.JsonTxArgs{},
			"TransactionArgs{From:<nil>, To:<nil>, Gas:<nil>, Nonce:<nil>, Data:<nil>, Input:<nil>, AccessList:<nil>}",
		},
		{
			"tx args with fields",
			types.JsonTxArgs{
				From:       &suite.addr,
				To:         &suite.addr,
				Gas:        &suite.hexUint64,
				Nonce:      &suite.hexUint64,
				Input:      &suite.hexInputBytes,
				Data:       &suite.hexDataBytes,
				AccessList: &ethcoretypes.AccessList{},
			},
			fmt.Sprintf("TransactionArgs{From:%v, To:%v, Gas:%v, Nonce:%v, Data:%v, Input:%v, AccessList:%v}",
				&suite.addr,
				&suite.addr,
				&suite.hexUint64,
				&suite.hexUint64,
				&suite.hexDataBytes,
				&suite.hexInputBytes,
				&ethcoretypes.AccessList{}),
		},
	}
	for _, tc := range testCases {
		outputString := tc.txArgs.String()
		suite.Require().Equal(outputString, tc.expectedString)
	}
}

func (suite *TxDataTestSuite) TestConvertTxArgsEthTx() {
	testCases := []struct {
		name   string
		txArgs types.JsonTxArgs
	}{
		{
			"empty tx args",
			types.JsonTxArgs{},
		},
		{
			"no nil args",
			types.JsonTxArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         &suite.hexBigInt,
				MaxPriorityFeePerGas: &suite.hexBigInt,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethcoretypes.AccessList{{Address: suite.addr, StorageKeys: []ethcommon.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
		},
		{
			"max fee per gas nil, but access list not nil",
			types.JsonTxArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         nil,
				MaxPriorityFeePerGas: &suite.hexBigInt,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethcoretypes.AccessList{{Address: suite.addr, StorageKeys: []ethcommon.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
		},
	}
	for _, tc := range testCases {
		res := tc.txArgs.ToTransaction()
		suite.Require().NotNil(res)
	}
}

func (suite *TxDataTestSuite) TestToMessageEVM() {
	testCases := []struct {
		name         string
		txArgs       types.JsonTxArgs
		globalGasCap uint64
		baseFee      *big.Int
		expError     bool
	}{
		{
			"empty tx args",
			types.JsonTxArgs{},
			uint64(0),
			nil,
			false,
		},
		{
			"specify gasPrice and (maxFeePerGas or maxPriorityFeePerGas)",
			types.JsonTxArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         &suite.hexBigInt,
				MaxPriorityFeePerGas: &suite.hexBigInt,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethcoretypes.AccessList{{Address: suite.addr, StorageKeys: []ethcommon.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			uint64(0),
			nil,
			true,
		},
		{
			"non-1559 execution, zero gas cap",
			types.JsonTxArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         nil,
				MaxPriorityFeePerGas: nil,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethcoretypes.AccessList{{Address: suite.addr, StorageKeys: []ethcommon.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			uint64(0),
			nil,
			false,
		},
		{
			"non-1559 execution, nonzero gas cap",
			types.JsonTxArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         nil,
				MaxPriorityFeePerGas: nil,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethcoretypes.AccessList{{Address: suite.addr, StorageKeys: []ethcommon.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			uint64(1),
			nil,
			false,
		},
		{
			"1559-type execution, nil gas price",
			types.JsonTxArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             nil,
				MaxFeePerGas:         &suite.hexBigInt,
				MaxPriorityFeePerGas: &suite.hexBigInt,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethcoretypes.AccessList{{Address: suite.addr, StorageKeys: []ethcommon.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			uint64(1),
			suite.bigInt,
			false,
		},
		{
			"1559-type execution, non-nil gas price",
			types.JsonTxArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         nil,
				MaxPriorityFeePerGas: nil,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethcoretypes.AccessList{{Address: suite.addr, StorageKeys: []ethcommon.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			uint64(1),
			suite.bigInt,
			false,
		},
	}
	for _, tc := range testCases {
		res, err := tc.txArgs.ToMessage(tc.globalGasCap, tc.baseFee)

		if tc.expError {
			suite.Require().NotNil(err)
		} else {
			suite.Require().Nil(err)
			suite.Require().NotNil(res)
		}
	}
}

func (suite *TxDataTestSuite) TestGetFrom() {
	testCases := []struct {
		name       string
		txArgs     types.JsonTxArgs
		expAddress ethcommon.Address
	}{
		{
			"empty from field",
			types.JsonTxArgs{},
			ethcommon.Address{},
		},
		{
			"non-empty from field",
			types.JsonTxArgs{
				From: &suite.addr,
			},
			suite.addr,
		},
	}
	for _, tc := range testCases {
		retrievedAddress := tc.txArgs.GetFrom()
		suite.Require().Equal(retrievedAddress, tc.expAddress)
	}
}

func (suite *TxDataTestSuite) TestGetData() {
	testCases := []struct {
		name           string
		txArgs         types.JsonTxArgs
		expectedOutput []byte
	}{
		{
			"empty input and data fields",
			types.JsonTxArgs{
				Data:  nil,
				Input: nil,
			},
			nil,
		},
		{
			"empty input field, non-empty data field",
			types.JsonTxArgs{
				Data:  &suite.hexDataBytes,
				Input: nil,
			},
			[]byte("data"),
		},
		{
			"non-empty input and data fields",
			types.JsonTxArgs{
				Data:  &suite.hexDataBytes,
				Input: &suite.hexInputBytes,
			},
			[]byte("input"),
		},
	}
	for _, tc := range testCases {
		retrievedData := tc.txArgs.GetData()
		suite.Require().Equal(retrievedData, tc.expectedOutput)
	}
}
