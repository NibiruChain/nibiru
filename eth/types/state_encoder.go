package types

import (
	fmt "fmt"

	"github.com/NibiruChain/collections"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// BytesToHex converts a byte array to a hexadecimal string
func BytesToHex(bz []byte) string {
	return fmt.Sprintf("%x", bz)
}

// EthAddr: (alias) 20 byte address of an Ethereum account.
type EthAddr = ethcommon.Address

// EthHash: (alias) 32 byte Keccak256 hash of arbitrary data.
type EthHash = ethcommon.Hash

var (
	// Implements a `collections.ValueEncoder` for the `[]byte` type
	ValueEncoderBytes collections.ValueEncoder[[]byte] = veBytes{}
	KeyEncoderBytes   collections.KeyEncoder[[]byte]   = keBytes{}

	// Implements a `collections.ValueEncoder` for an Ethereum address.
	ValueEncoderEthAddr collections.ValueEncoder[EthAddr] = veEthAddr{}
	// keEthHash: Implements a `collections.KeyEncoder` for an Ethereum address.
	KeyEncoderEthAddr collections.KeyEncoder[EthAddr] = keEthAddr{}

	// keEthHash: Implements a `collections.KeyEncoder` for an Ethereum hash.
	KeyEncoderEthHash collections.KeyEncoder[EthHash] = keEthHash{}
)

// collections ValueEncoder[[]byte]
type veBytes struct{}

func (_ veBytes) Encode(value []byte) []byte    { return value }
func (_ veBytes) Decode(bz []byte) []byte       { return bz }
func (_ veBytes) Stringify(value []byte) string { return BytesToHex(value) }
func (_ veBytes) Name() string                  { return "[]byte" }

// veEthAddr: Implements a `collections.ValueEncoder` for an Ethereum address.
type veEthAddr struct{}

func (_ veEthAddr) Encode(value EthAddr) []byte    { return value.Bytes() }
func (_ veEthAddr) Decode(bz []byte) EthAddr       { return ethcommon.BytesToAddress(bz) }
func (_ veEthAddr) Stringify(value EthAddr) string { return value.Hex() }
func (_ veEthAddr) Name() string                   { return "EthAddr" }

type keBytes struct{}

// Encode encodes the type T into bytes.
func (_ keBytes) Encode(key []byte) []byte { return key }

// Decode decodes the given bytes back into T.
// And it also must return the bytes of the buffer which were read.
func (_ keBytes) Decode(bz []byte) (int, []byte) { return len(bz), bz }

// Stringify returns a string representation of T.
func (_ keBytes) Stringify(key []byte) string { return BytesToHex(key) }

// keEthAddr: Implements a `collections.KeyEncoder` for an Ethereum address.
type keEthAddr struct{}

func (_ keEthAddr) Encode(value EthAddr) []byte { return value.Bytes() }
func (_ keEthAddr) Decode(bz []byte) (int, EthAddr) {
	return ethcommon.AddressLength, ethcommon.BytesToAddress(bz)
}
func (_ keEthAddr) Stringify(value EthAddr) string { return value.Hex() }

// keEthHash: Implements a `collections.KeyEncoder` for an Ethereum hash.
type keEthHash struct{}

func (_ keEthHash) Encode(value EthHash) []byte { return value.Bytes() }
func (_ keEthHash) Decode(bz []byte) (int, EthHash) {
	return ethcommon.HashLength, ethcommon.BytesToHash(bz)
}
func (_ keEthHash) Stringify(value EthHash) string { return value.Hex() }
