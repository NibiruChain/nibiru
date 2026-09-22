package impl

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/cometbft/cometbft/crypto/tmhash"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
)

const (
	txHashHexLength = tmhash.Size * 2

	txHashFormat = "expected a bare 64-character hexadecimal CometBFT hash or a 0x-prefixed 64-character hexadecimal EVM hash"
)

// TxHash is the closed set of transaction-hash representations accepted by
// the CLI. Its concrete value is either CometBFTTxHash or EVMTxHash.
type TxHash interface {
	Canonical() string
	isTxHash()
}

var (
	_ TxHash = CometBFTTxHash{}
	_ TxHash = EVMTxHash{}
)

// CometBFTTxHash is the fixed-size digest returned by types.Tx.Hash.
type CometBFTTxHash struct {
	bytes [tmhash.Size]byte
}

// EVMTxHash is an Ethereum transaction hash backed by geth's common.Hash.
type EVMTxHash struct {
	hash gethcommon.Hash
}

// ErrInvalidTxHash indicates that a transaction hash failed syntax validation
// before any backend query can run.
var ErrInvalidTxHash = errors.New("invalid transaction hash")

// ClassifyTxHash validates and classifies a user-supplied transaction hash.
//
// A bare 64-character hexadecimal value uses the CometBFT spelling. An EVM
// hash must use the 0x-prefixed spelling so the two families have distinct
// CLI syntax. This function validates syntax only. It does not check whether
// the transaction exists or is indexed.
func ClassifyTxHash(input string) (TxHash, error) {
	if input == "" {
		return nil, invalidTxHash("input is empty")
	}
	if strings.TrimSpace(input) != input {
		return nil, invalidTxHash("input contains leading or trailing whitespace")
	}

	isEVM := false
	hexValue := input
	if len(input) >= 2 && (input[:2] == "0x" || input[:2] == "0X") {
		isEVM = true
		hexValue = input[2:]
	}

	if len(hexValue) != txHashHexLength {
		return nil, invalidTxHash("input must contain exactly 32 bytes")
	}

	var digest [tmhash.Size]byte
	if _, err := hex.Decode(digest[:], []byte(hexValue)); err != nil {
		return nil, invalidTxHash("input contains non-hexadecimal characters")
	}

	if isEVM {
		var hash gethcommon.Hash
		copy(hash[:], digest[:])
		return EVMTxHash{hash: hash}, nil
	}

	return CometBFTTxHash{bytes: digest}, nil
}

func (h CometBFTTxHash) isTxHash() {}

// Bytes returns a copy of the 32-byte CometBFT transaction hash.
func (h CometBFTTxHash) Bytes() []byte {
	return append([]byte(nil), h.bytes[:]...)
}

// Canonical returns uppercase hexadecimal without a 0x prefix, matching the
// repository's CometBFT hash formatter.
func (h CometBFTTxHash) Canonical() string {
	return eth.TmTxHashToString(h.bytes[:])
}

func (h CometBFTTxHash) String() string { return h.Canonical() }

func (h EVMTxHash) isTxHash() {}

// Hash returns the validated geth common.Hash value.
func (h EVMTxHash) Hash() gethcommon.Hash { return h.hash }

// Canonical returns the 0x-prefixed lowercase hexadecimal spelling generated
// by geth's common.Hash.Hex method.
func (h EVMTxHash) Canonical() string { return h.hash.Hex() }

func (h EVMTxHash) String() string { return h.Canonical() }

func invalidTxHash(reason string) error {
	return fmt.Errorf("%w: %s; %s", ErrInvalidTxHash, reason, txHashFormat)
}
