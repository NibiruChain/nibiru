package collections

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultSequenceStart is the initial starting number of the Sequence.
const DefaultSequenceStart uint64 = 0

// Sequence defines a collection item which contains an always increasing number.
// Useful for those flows which require ever raising unique ids.
type Sequence struct {
	sequence Item[uint64]
}

// NewSequence instantiates a new sequence object.
func NewSequence(sk types.StoreKey, namespace Namespace) Sequence {
	return Sequence{
		sequence: NewItem[uint64](sk, namespace, uint64Value{}),
	}
}

// Next returns the next available sequence number
// and also increases the sequence number count.
func (s Sequence) Next(ctx sdk.Context) uint64 {
	// get current
	seq := s.Peek(ctx)
	// increase
	s.sequence.Set(ctx, seq+1)
	// return current
	return seq
}

// Peek returns current sequence number without increasing it.
func (s Sequence) Peek(ctx sdk.Context) uint64 {
	return s.sequence.GetOr(ctx, DefaultSequenceStart)
}

// IncreaseBy increases the sequence by some value and retruns the new number.
func (s Sequence) IncreaseBy(ctx sdk.Context, by uint64) (nextVal uint64) {
	curr := s.Peek(ctx)
	nextVal = curr + by
	s.Set(ctx, nextVal)
	return nextVal
}

// Set hard resets the sequence to the provided number.
func (s Sequence) Set(ctx sdk.Context, u uint64) {
	s.sequence.Set(ctx, u)
}

// uint64Value implements a ValueEncoder for uint64
type uint64Value struct{}

func (u uint64Value) Encode(value uint64) []byte    { return sdk.Uint64ToBigEndian(value) }
func (u uint64Value) Decode(b []byte) uint64        { return sdk.BigEndianToUint64(b) }
func (u uint64Value) Stringify(value uint64) string { return strconv.FormatUint(value, 10) }
func (u uint64Value) Name() string                  { return "uint64" }
