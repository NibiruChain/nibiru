package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
)

type gaslessMeter struct {
	consumed types.Gas
}

// GasLessMeter returns a reference to a new gaslessMeter.
func GasLessMeter() types.GasMeter {
	return &gaslessMeter{
		consumed: 1,
	}
}

func (g *gaslessMeter) GasConsumed() types.Gas {
	return 1
}

func (g *gaslessMeter) GasConsumedToLimit() types.Gas {
	return 1
}

func (g *gaslessMeter) Limit() types.Gas {
	return 1
}

func (g *gaslessMeter) ConsumeGas(types.Gas, string) {}
func (g *gaslessMeter) RefundGas(types.Gas, string)  {}

func (g *gaslessMeter) IsPastLimit() bool {
	return false
}

func (g *gaslessMeter) IsOutOfGas() bool {
	return false
}

func (g *gaslessMeter) String() string {
	return fmt.Sprintf("GaslessMeter:\n  consumed: %d", g.consumed)
}
