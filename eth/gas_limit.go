// Copyright (c) 2023-2024 Nibi, Inc.
package eth

import (
	"encoding/json"
	fmt "fmt"
	math "math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BlockGasLimit: Gas (limit) as defined by the block gas meter. Gas limit is
// derived from the consensus params if the block gas meter is nil.
func BlockGasLimit(ctx sdk.Context) (gasLimit uint64) {
	blockGasMeter := ctx.BlockGasMeter()

	// Get the limit from the gas meter only if its not null and not an
	// InfiniteGasMeter
	if blockGasMeter != nil && blockGasMeter.Limit() != 0 {
		return blockGasMeter.Limit()
	}

	// Otherwise get from the consensus parameters
	cp := ctx.ConsensusParams()
	if cp == nil || cp.Block == nil {
		return 0
	}

	maxGas := cp.Block.MaxGas

	// Setting max_gas to -1 in Tendermint means there is no limit on the maximum gas consumption for transactions
	// https://github.com/cometbft/cometbft/blob/v0.37.2/proto/tendermint/types/params.proto#L25-L27
	if maxGas == -1 {
		return math.MaxUint64
	}

	if maxGas > 0 {
		return uint64(maxGas) // #nosec G701 -- maxGas is int64 type. It can never be greater than math.MaxUint64
	}

	return 0
}

// NewInfiniteGasMeterWithLimit returns a reference to a new infiniteGasMeter.
func NewInfiniteGasMeterWithLimit(limit sdk.Gas) sdk.GasMeter {
	return &InfiniteGasMeter{
		consumed: 0,
		limit:    limit,
	}
}

// NewInfiniteGasMeter: Alias for an infinite gas meter
// ([NewInfiniteGasMeterWithLimitla)] with a tracked but unenforced gas limit.
func NewInfiniteGasMeter() sdk.GasMeter {
	return NewInfiniteGasMeterWithLimit(math.MaxUint64)
}

var _ sdk.GasMeter = &InfiniteGasMeter{}

// InfiniteGasMeter: A special impl of `sdk.GasMeter` that ignores any gas
// limits, allowing an unlimited amount of gas to be consumed. This is especially
// useful for scenarios where gas consumption needs to be monitored but not
// restricted, such as during testing or in parts of the chain where constraints
// are meant to be set differently.
type InfiniteGasMeter struct {
	// consumed: Tracks the amount of gas units consumed.
	consumed sdk.Gas
	// limit: Nominal unit for the gas limit, which is not enforced in a way that
	// restricts consumption.
	limit sdk.Gas
}

// GasConsumedToLimit returns the gas limit if gas consumed is past the limit,
// otherwise it returns the consumed gas.
//
// Note that This function is used when recovering
// from a panic in "BlockGasMeter" when the consumed gas passes the limit.
func (g *InfiniteGasMeter) GasConsumedToLimit() sdk.Gas {
	return g.consumed
}

// GasConsumed returns the gas consumed from the GasMeter.
func (g *InfiniteGasMeter) GasConsumed() sdk.Gas {
	return g.consumed
}

// Limit returns the gas limit of the GasMeter.
func (g *InfiniteGasMeter) Limit() sdk.Gas {
	return g.limit
}

// addUint64Overflow performs the addition operation on two uint64 integers and
// returns a boolean on whether or not the result overflows.
func addUint64Overflow(a, b uint64) (uint64, bool) {
	if math.MaxUint64-a < b {
		return 0, true
	}

	return a + b, false
}

// ConsumeGas adds the given amount of gas to the gas consumed and panics if it overflows the limit or out of gas.
func (g *InfiniteGasMeter) ConsumeGas(amount sdk.Gas, descriptor string) {
	var overflow bool
	// TODO: Should we set the consumed field after overflow checking?
	g.consumed, overflow = addUint64Overflow(g.consumed, amount)
	if overflow {
		panic(ErrorGasOverflow{descriptor})
	}
}

// RefundGas will deduct the given amount from the gas consumed. If the amount is
// greater than the gas consumed, the function will panic.
//
// Use case: This functionality enables refunding gas to the trasaction or block gas pools so that
// EVM-compatible chains can fully support the go-ethereum StateDb interface.
// See https://github.com/cosmos/cosmos-sdk/pull/9403 for reference.
func (g *InfiniteGasMeter) RefundGas(amount sdk.Gas, descriptor string) {
	if g.consumed < amount {
		panic(ErrorNegativeGasConsumed{Descriptor: descriptor})
	}

	g.consumed -= amount
}

// IsPastLimit returns true if gas consumed is past limit, otherwise it returns
// false. In the case of the the [InfiniteGasMeter], this always returns false.
func (g *InfiniteGasMeter) IsPastLimit() bool {
	return false
}

// IsOutOfGas returns true if gas consumed is greater than or equal to gas limit,
// otherwise it returns false. In the case of the the [InfiniteGasMeter], this
// always returns false for unrestricted gas consumption.
func (g *InfiniteGasMeter) IsOutOfGas() bool {
	return false
}

// String returns the BasicGasMeter's gas limit and gas consumed.
func (g *InfiniteGasMeter) String() string {
	data := map[string]uint64{"consumed": g.consumed, "limit": g.limit}
	jsonData, _ := json.Marshal(data)
	return fmt.Sprintf("InfiniteGasMeter: %s", jsonData)
}

// GasRemaining returns MaxUint64 since limit is not confined in infiniteGasMeter.
func (g *InfiniteGasMeter) GasRemaining() sdk.Gas {
	return math.MaxUint64
}

// ErrorNegativeGasConsumed defines an error thrown when the amount of gas
// refunded results in a negative gas consumed amount.
type ErrorNegativeGasConsumed struct {
	Descriptor string
}

// ErrorGasOverflow defines an error thrown when an action results gas consumption
// unsigned integer overflow.
type ErrorGasOverflow struct {
	Descriptor string
}
