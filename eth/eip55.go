package eth

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

var _ sdk.CustomProtobufType = (*EIP55Addr)(nil)

// EIP55Addr is a wrapper around gethcommon.Address that provides JSON marshaling
// and unmarshalling as well as Protobuf serialization and deserialization.
// The constructors ensure that the input string is a valid 20 byte hex address.
type EIP55Addr struct {
	gethcommon.Address
}

// Checks input length, but not case-sensitive hex.
func NewEIP55AddrFromStr(input string) (EIP55Addr, error) {
	if !gethcommon.IsHexAddress(input) {
		return EIP55Addr{}, fmt.Errorf(
			"EIP55AddrError: input \"%s\" is not an ERC55-compliant, 20 byte hex address",
			input,
		)
	}

	addr := EIP55Addr{
		Address: gethcommon.HexToAddress(input),
	}

	return addr, nil
}

// Marshal implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h EIP55Addr) Marshal() ([]byte, error) {
	return []byte(h.Address.Hex()), nil
}

// MarshalJSON returns the [EIP55Addr] as JSON bytes.
// Implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h EIP55Addr) MarshalJSON() ([]byte, error) {
	return []byte(h.String()), nil
}

// MarshalTo serializes a EIP55Addr directly into a pre-allocated byte slice ("data").
// MarshalTo implements the gogo proto custom type interface.
// Implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h *EIP55Addr) MarshalTo(data []byte) (n int, err error) {
	copy(data, []byte(h.Address.Hex()))
	return h.Size(), nil
}

// Unmarshal implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h *EIP55Addr) Unmarshal(data []byte) error {
	fmt.Printf("Unmarshal data: %s\n", data)
	addr, err := NewEIP55AddrFromStr(string(data))
	if err != nil {
		return err
	}
	*h = addr
	return nil
}

// UnmarshalJSON implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h *EIP55Addr) UnmarshalJSON(bz []byte) error {
	addr, err := NewEIP55AddrFromStr(string(bz))
	if err != nil {
		return err
	}
	*h = addr
	return nil
}

// Size implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h EIP55Addr) Size() int {
	return len([]byte(h.Address.Hex()))
}
