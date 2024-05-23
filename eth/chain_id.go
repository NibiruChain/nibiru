// Copyright (c) 2023-2024 Nibi, Inc.
package eth

import (
	"math/big"

	"github.com/NibiruChain/nibiru/app/appconst"
)

// IsValidChainID returns false if the given chain identifier is incorrectly
// formatted.
func IsValidChainID(chainID string) bool {
	return len(chainID) <= 48
}

// ParseEthChainID parses a string chain identifier's epoch to an
// Ethereum-compatible chain-id in *big.Int format.
//
// This function uses Nibiru's map of chain IDs defined in Nibiru/app/appconst
// rather than the regex of EIP155, which is implemented by
// ParseEthChainIDStrict.
func ParseEthChainID(chainID string) (*big.Int, error) {
	return appconst.GetEthChainID(chainID), nil
}

// ParseEthChainIDStrict parses a string chain identifier's epoch to an
// Ethereum-compatible chain-id in *big.Int format. The function returns an error
// if the chain-id has an invalid format
func ParseEthChainIDStrict(chainID string) (*big.Int, error) {
	return appconst.GetEthChainID(chainID), nil
}
