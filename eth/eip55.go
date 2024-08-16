package eth

import (
	"encoding/json"
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
	return h.Bytes(), nil
}

// MarshalJSON returns the [EIP55Addr] as JSON bytes.
// Implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h EIP55Addr) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.String())
}

// MarshalTo serializes a EIP55Addr directly into a pre-allocated byte slice ("data").
// MarshalTo implements the gogo proto custom type interface.
// Implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h *EIP55Addr) MarshalTo(data []byte) (n int, err error) {
	copy(data, h.Bytes())
	return h.Size(), nil
}

// Unmarshal implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h *EIP55Addr) Unmarshal(data []byte) error {
	addr := gethcommon.BytesToAddress(data)
	*h = EIP55Addr{Address: addr}
	return nil
}

// UnmarshalJSON implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h *EIP55Addr) UnmarshalJSON(bz []byte) error {
	text := new(string)
	if err := json.Unmarshal(bz, text); err != nil {
		return err
	}

	addr, err := NewEIP55AddrFromStr(*text)
	if err != nil {
		return err
	}

	*h = addr

	return nil
}

// Size implements the gogo proto custom type interface.
// Ref: https://github.com/cosmos/gogoproto/blob/v1.5.0/custom_types.md
func (h EIP55Addr) Size() int {
	return len(h.Bytes())
}
