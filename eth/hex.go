package eth

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

/////////// HexAddr

// HexAddr: An ERC55-compliant hexadecimal-encoded string representing the 20
// byte address of an Ethereum account.
type HexAddr string

var _ sdk.CustomProtobufType = (*HexAddr)(nil)

func NewHexAddr(addr gethcommon.Address) HexAddr {
	return HexAddr(addr.Hex())
}

func NewHexAddrFromStr(addr string) (HexAddr, error) {
	hexAddr := HexAddr(gethcommon.HexToAddress(addr).Hex())
	if !gethcommon.IsHexAddress(addr) {
		return hexAddr, fmt.Errorf(
			"%s: input \"%s\" is not an ERC55-compliant, 20 byte hex address",
			HexAddrError, addr,
		)
	}
	return hexAddr, hexAddr.Valid()
}

// MustNewHexAddrFromStr is the same as [NewHexAddrFromStr], except it panics
// when there's an error.
func MustNewHexAddrFromStr(addr string) HexAddr {
	hexAddr, err := NewHexAddrFromStr(addr)
	if err != nil {
		panic(err)
	}
	return hexAddr
}

const HexAddrError = "HexAddrError"

func (h HexAddr) Valid() error {
	// Check address encoding bijectivity
	wantAddr := h.ToAddr().Hex() // gethcommon.Address.Hex()
	haveAddr := string(h)        // should be equivalent to ↑

	if !gethcommon.IsHexAddress(haveAddr) || haveAddr != wantAddr {
		return fmt.Errorf(
			"%s: Ethereum address is not represented as expected. We have encoding \"%s\" and instead need \"%s\" (gethcommon.Address.Hex)",
			HexAddrError, haveAddr, wantAddr,
		)
	}

	return nil
}

func (h HexAddr) ToAddr() gethcommon.Address {
	return gethcommon.HexToAddress(string(h))
}

// ToBytes gets the string representation of the underlying address.
func (h HexAddr) ToBytes() []byte {
	return h.ToAddr().Bytes()
}

func (h HexAddr) String() string { return h.ToAddr().Hex() }

// Marshal implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h HexAddr) Marshal() ([]byte, error) {
	return []byte(h), nil
}

// MarshalJSON returns the [HexAddr] as JSON bytes.
// Implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h HexAddr) MarshalJSON() ([]byte, error) {
	return []byte("\"" + h.String() + "\""), nil // a string is already JSON
}

// MarshalTo serializes a pre-allocated byte slice ("data") directly into the
// [HexAddr] value, avoiding unnecessary memory allocations.
// MarshalTo implements the gogo proto custom type interface.
// Implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h *HexAddr) MarshalTo(data []byte) (n int, err error) {
	bz := []byte(*h)
	copy(data, bz)
	hexAddr, err := NewHexAddrFromStr(string(bz))
	*h = hexAddr
	return h.Size(), err
}

// Unmarshal implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h *HexAddr) Unmarshal(data []byte) error {
	hexAddr, err := NewHexAddrFromStr(string(data))
	*h = hexAddr
	return err
}

// UnmarshalJSON implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h *HexAddr) UnmarshalJSON(bz []byte) error {
	text := new(string)
	if err := json.Unmarshal(bz, text); err != nil {
		return err
	}

	hexAddr, err := NewHexAddrFromStr(*text)
	if err != nil {
		return err
	}

	*h = hexAddr

	return nil
}

// Size implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h HexAddr) Size() int {
	return len(h)
}
