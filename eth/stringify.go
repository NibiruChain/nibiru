package eth

import (
	"encoding/hex"
	fmt "fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
)

// TmTxHashToString returns the consensus transaction hash as a string.
// Transactions are hex-encoded and capitlized.
// Reference: Tx.String function from comet-bft/types/tx.go
func TmTxHashToString(tmTxHash []byte) string {
	return fmt.Sprintf("%X", tmTxHash)
}

// EthTxHashToString returns the EVM transaction hash as a string.
func EthTxHashToString(hash gethcommon.Hash) string {
	return hash.Hex()
}

// BloomToHex returns the bloom filter as a string.
func BloomToHex(bloom gethcore.Bloom) string {
	return BytesToHex(bloom.Bytes())
}

// BloomFromHex converts a hex-encoded bloom filter to a gethcore.Bloom.
func BloomFromHex(bloomHex string) (gethcore.Bloom, error) {
	bloomBz, err := hex.DecodeString(bloomHex)
	if err != nil {
		return gethcore.Bloom{}, fmt.Errorf("could not construct bloom: %w", err)
	}
	return gethcore.BytesToBloom(bloomBz), nil
}
