// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"

	"github.com/ethereum/go-ethereum/params"
)

// EthereumConfig returns an Ethereum ChainConfig for EVM state transitions.
func EthereumConfig(chainID *big.Int) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:             chainID,
		HomesteadBlock:      Big0,
		DAOForkBlock:        Big0,
		DAOForkSupport:      true,
		EIP150Block:         Big0,
		EIP155Block:         Big0,
		EIP158Block:         Big0,
		ByzantiumBlock:      Big0,
		ConstantinopleBlock: Big0,
		PetersburgBlock:     Big0,
		IstanbulBlock:       Big0,
		MuirGlacierBlock:    Big0,
		BerlinBlock:         Big0,
		LondonBlock:         Big0,
		ArrowGlacierBlock:   Big0,
		GrayGlacierBlock:    Big0,
		MergeNetsplitBlock:  Big0,
		// Shanghai switch time (nil = no fork, 0 => already on shanghai)
		ShanghaiTime: ptrU64(0),
		// CancunTime switch time (nil = no fork, 0 => already on cancun)
		CancunTime:              nil, // nil => disable "blobs"
		PragueTime:              nil, // nil => disable EIP-7702, blob improvements, and increased CALL gas costs
		VerkleTime:              nil, // nil => disable stateless verification
		TerminalTotalDifficulty: nil,
		Ethash:                  nil,
		Clique:                  nil,
	}
}

func ptrU64(n uint) *uint64 {
	u64 := uint64(n)
	return &u64
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
