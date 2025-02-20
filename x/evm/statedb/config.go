package statedb

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// TxConfig encapsulates the readonly information of current tx for `StateDB`.
type TxConfig struct {
	BlockHash gethcommon.Hash // hash of current block
	TxHash    gethcommon.Hash // hash of current tx
	TxIndex   uint            // the index of current transaction
	LogIndex  uint            // the index of next log within current block
}

// NewEmptyTxConfig construct an empty TxConfig,
// used in context where there's no transaction, e.g. `eth_call`/`eth_estimateGas`.
func NewEmptyTxConfig(blockHash gethcommon.Hash) TxConfig {
	return TxConfig{
		BlockHash: blockHash,
		TxHash:    gethcommon.Hash{},
		TxIndex:   0,
		LogIndex:  0,
	}
}

// EVMConfig encapsulates parameters needed to create an instance of the EVM
// ("go-ethereum/core/vm.EVM").
type EVMConfig struct {
	Params      evm.Params
	ChainConfig *gethparams.ChainConfig

	// BlockCoinbase: In Ethereum, the coinbase (or "benficiary") is the address that
	// proposed the current block. It corresponds to the [COINBASE op code]
	// (the "block.coinbase" stack output).
	//
	// [COINBASE op code]: https://ethereum.org/en/developers/docs/evm/opcodes/
	BlockCoinbase gethcommon.Address

	// BaseFeeWei is the EVM base fee in units of wei per gas. The term "base
	// fee" comes from EIP-1559.
	BaseFeeWei *big.Int
}
