package evm

import (
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
)

var MOCK_GETH_MESSAGE = core.Message{
	To:               nil,
	From:             EVM_MODULE_ADDRESS,
	Nonce:            0,
	Value:            Big0, // amount
	GasLimit:         0,
	GasPrice:         Big0,
	GasFeeCap:        Big0,
	GasTipCap:        Big0,
	Data:             []byte{},
	AccessList:       gethcore.AccessList{},
	SkipNonceChecks:  false,
	SkipFromEOACheck: false,
}
