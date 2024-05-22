// Copyright (c) 2023-2024 Nibi, Inc.
package eth

import (
	"fmt"
	"math/big"
	"regexp"

	"github.com/NibiruChain/nibiru/app/appconst"
)

var (
	// one of any lower case letter from "a"-"z"
	regexChainID = `[a-z]{1,}`
	// one of either "_" or "-"
	regexEIP155Separator = `[_-]{1}`
	// one of "_"
	// regexEIP155Separator = `_{1}`
	regexEIP155         = `[1-9][0-9]*`
	regexEpochSeparator = `-{1}`
	regexEpoch          = `[1-9][0-9]*`
	nibiruEvmChainId    = regexp.MustCompile(fmt.Sprintf(`^(%s)%s(%s)%s(%s)$`,
		regexChainID,
		regexEIP155Separator,
		regexEIP155,
		regexEpochSeparator,
		regexEpoch))
)

// IsValidChainID returns false if the given chain identifier is incorrectly
// formatted.
func IsValidChainID(chainID string) bool {
	if len(chainID) > 48 {
		return false
	}

	return nibiruEvmChainId.MatchString(chainID)
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
	evmChainID, exists := appconst.EVMChainIDs[chainID]
	if exists {
		return evmChainID, nil
	} else {
		return appconst.DefaultEVMChainID, nil
	}
}
