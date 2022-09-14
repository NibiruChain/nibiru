package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	wellknown "github.com/gogo/protobuf/types"
)

// DefaultSequenceStart is the initial starting number of the Sequence.
const DefaultSequenceStart uint64 = 1

// Sequence defines a collection item which contains an always increasing number.
// Useful for those flows which require ever raising unique ids.
type Sequence struct {
	sequence Item[wellknown.UInt64Value, *wellknown.UInt64Value]
}

// NewSequence instantiates a new sequence object.
func NewSequence(cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8) Sequence {
	return Sequence{
		sequence: NewItem[wellknown.UInt64Value](cdc, sk, prefix),
	}
}

// Next returns the next available sequence number
// and also increases the sequence number count.
func (s Sequence) Next(ctx sdk.Context) uint64 {
	// get current
	seq := s.Peek(ctx)
	// increase
	s.sequence.Set(ctx, wellknown.UInt64Value{Value: seq + 1})
	// return current
	return seq
}

// Peek gets the next available sequence number without increasing it.
func (s Sequence) Peek(ctx sdk.Context) uint64 {
	return s.sequence.GetOr(ctx, wellknown.UInt64Value{Value: DefaultSequenceStart}).Value
}

// Set hard resets the sequence to the provided number.
func (s Sequence) Set(ctx sdk.Context, u uint64) {
	s.sequence.Set(ctx, wellknown.UInt64Value{Value: u})
}
