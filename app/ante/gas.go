package ante

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cosmos/cosmos-sdk/types"
)

type fixedGasMeter struct {
	consumed types.Gas
}

// NewFixedGasMeter returns a reference to a new fixedGasMeter.
func NewFixedGasMeter(fixedGas types.Gas) types.GasMeter {
	return &fixedGasMeter{
		consumed: fixedGas,
	}
}

func (g *fixedGasMeter) GasConsumed() types.Gas {
	return g.consumed
}

func (g *fixedGasMeter) GasConsumedToLimit() types.Gas {
	return g.consumed
}

func (g *fixedGasMeter) Limit() types.Gas {
	return g.consumed
}

func (g *fixedGasMeter) GasRemaining() storetypes.Gas {
	return g.consumed
}

func (g *fixedGasMeter) ConsumeGas(types.Gas, string) {}
func (g *fixedGasMeter) RefundGas(types.Gas, string)  {}

func (g *fixedGasMeter) IsPastLimit() bool {
	return false
}

func (g *fixedGasMeter) IsOutOfGas() bool {
	return false
}

func (g *fixedGasMeter) String() string {
	return fmt.Sprintf("GaslessMeter:\n  consumed: %d", g.consumed)
}
