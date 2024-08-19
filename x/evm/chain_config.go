// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// EthereumConfig returns an Ethereum ChainConfig for EVM state transitions.
func EthereumConfig(chainID *big.Int) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:                 chainID,
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            big.NewInt(0),
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(0),
		EIP150Hash:              common.Hash{},
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		GrayGlacierBlock:        big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		ShanghaiBlock:           nil, // TODO: change this if we upgrade go-ethereum dependency
		CancunBlock:             nil, // TODO: change this if we upgrade go-ethereum dependency
		TerminalTotalDifficulty: nil,
		Ethash:                  nil,
		Clique:                  nil,
	}
}

// Validate performs a basic validation of the ChainConfig params. The function will return an error
// if any of the block values is uninitialized (i.e. nil) or if the EIP150Hash is an invalid hash.
func Validate() error {
	// NOTE: chain ID is not needed to check config order
	if err := EthereumConfig(nil).CheckConfigForkOrder(); err != nil {
		return errorsmod.Wrap(err, "invalid config fork order")
	}
	return nil
}
