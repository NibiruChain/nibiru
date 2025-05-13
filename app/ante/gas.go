package ante

import (
	"fmt"

	"cosmossdk.io/store"
)

type fixedGasMeter struct {
	consumed store.Gas
}

// NewFixedGasMeter returns a reference to a new fixedGasMeter.
func NewFixedGasMeter(fixedGas store.Gas) store.GasMeter {
	return &fixedGasMeter{
		consumed: fixedGas,
	}
}

func (g *fixedGasMeter) GasConsumed() store.Gas {
	return g.consumed
}

func (g *fixedGasMeter) GasConsumedToLimit() store.Gas {
	return g.consumed
}

func (g *fixedGasMeter) Limit() store.Gas {
	return g.consumed
}

func (g *fixedGasMeter) GasRemaining() store.Gas {
	return g.consumed
}

// ConsumeGas is a no-op because the fixed gas meter stays fixed.
func (g *fixedGasMeter) ConsumeGas(store.Gas, string) {}

// RefundGas is a no-op because the fixed gas meter stays fixed.
func (g *fixedGasMeter) RefundGas(store.Gas, string) {}

func (g *fixedGasMeter) IsPastLimit() bool {
	return false
}

func (g *fixedGasMeter) IsOutOfGas() bool {
	return false
}

func (g *fixedGasMeter) String() string {
	return fmt.Sprintf("GaslessMeter:\n  consumed: %d", g.consumed)
}
