package eth

import (
	"encoding/hex"
	fmt "fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/common/hexutil"
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

// TODO: test with real bloom
// BloomToString returns bloom filter as a string.
func BloomToString(bloom gethcore.Bloom) string {
	return BytesToHex(bloom.Bytes())
}

// TODO: test with real bloom
// BloomFromString returns bloom filter as a string.
func BloomFromString(bloomHex string) (gethcore.Bloom, error) {
	bloomBz, err := hex.DecodeString(bloomHex)
	if err != nil {
		return gethcore.Bloom{}, fmt.Errorf("could not construct bloom: %w", err)
	}
	return gethcore.BytesToBloom(bloomBz), nil
}
