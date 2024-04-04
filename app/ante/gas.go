package ante

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
)

type fixedGasMeter struct {
	consumed storetypes.Gas
}

// NewFixedGasMeter returns a reference to a new fixedGasMeter.
func NewFixedGasMeter(fixedGas storetypes.Gas) storetypes.GasMeter {
	return &fixedGasMeter{
		consumed: fixedGas,
	}
}

func (g *fixedGasMeter) GasConsumed() storetypes.Gas {
	return g.consumed
}

func (g *fixedGasMeter) GasConsumedToLimit() storetypes.Gas {
	return g.consumed
}

func (g *fixedGasMeter) Limit() storetypes.Gas {
	return g.consumed
}

func (g *fixedGasMeter) GasRemaining() storetypes.Gas {
	return g.consumed
}

func (g *fixedGasMeter) ConsumeGas(storetypes.Gas, string) {}
func (g *fixedGasMeter) RefundGas(storetypes.Gas, string)  {}

func (g *fixedGasMeter) IsPastLimit() bool {
	return false
}

func (g *fixedGasMeter) IsOutOfGas() bool {
	return false
}

func (g *fixedGasMeter) String() string {
	return fmt.Sprintf("GaslessMeter:\n  consumed: %d", g.consumed)
}
