package ante

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
)

type fixedGasMeter struct {
	consumed uint64
}

// NewFixedGasMeter returns a reference to a new fixedGasMeter.
func NewFixedGasMeter(fixedGas uint64) storetypes.GasMeter {
	return &fixedGasMeter{
		consumed: fixedGas,
	}
}

func (g *fixedGasMeter) GasConsumed() uint64 {
	return g.consumed
}

func (g *fixedGasMeter) GasConsumedToLimit() uint64 {
	return g.consumed
}

func (g *fixedGasMeter) Limit() uint64 {
	return g.consumed
}

func (g *fixedGasMeter) GasRemaining() storetypes.Gas {
	return g.consumed
}

// ConsumeGas is a no-op because the fixed gas meter stays fixed.
func (g *fixedGasMeter) ConsumeGas(uint64, string) {}

// RefundGas is a no-op because the fixed gas meter stays fixed.
func (g *fixedGasMeter) RefundGas(uint64, string) {}

func (g *fixedGasMeter) IsPastLimit() bool {
	return false
}

func (g *fixedGasMeter) IsOutOfGas() bool {
	return false
}

func (g *fixedGasMeter) String() string {
	return fmt.Sprintf("GaslessMeter:\n  consumed: %d", g.consumed)
}
