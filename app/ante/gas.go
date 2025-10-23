package ante

import (
	"fmt"
	"math"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type fixedGasMeter struct {
	consumed sdk.Gas
}

// NewFixedGasMeter returns a reference to a new fixedGasMeter.
func NewFixedGasMeter(fixedGas sdk.Gas) sdk.GasMeter {
	return &fixedGasMeter{
		consumed: fixedGas,
	}
}

func (g *fixedGasMeter) GasConsumed() sdk.Gas {
	return g.consumed
}

func (g *fixedGasMeter) GasConsumedToLimit() sdk.Gas {
	return g.consumed
}

func (g *fixedGasMeter) Limit() sdk.Gas {
	return math.MaxUint64 // unconfined
}

func (g *fixedGasMeter) GasRemaining() storetypes.Gas {
	return math.MaxUint64 // unconfined
}

// ConsumeGas is a no-op because the fixed gas meter stays fixed.
func (g *fixedGasMeter) ConsumeGas(sdk.Gas, string) {}

// RefundGas is a no-op because the fixed gas meter stays fixed.
func (g *fixedGasMeter) RefundGas(sdk.Gas, string) {}

// IsPastLimit returns false since the gas limit is not confined.
func (g *fixedGasMeter) IsPastLimit() bool {
	return false // unconfined
}

// IsOutOfGas returns false since the gas limit is not confined.
func (g *fixedGasMeter) IsOutOfGas() bool {
	return false // unconfined
}

func (g *fixedGasMeter) String() string {
	return fmt.Sprintf("GaslessMeter:\n  consumed: %d", g.consumed)
}
