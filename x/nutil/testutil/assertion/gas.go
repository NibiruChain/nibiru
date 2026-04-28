package assertion

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GasConsumedShouldBe(ctx sdk.Context, gasConsumed uint64) error {
	gasUsed := ctx.GasMeter().GasConsumed()
	if gasConsumed != gasUsed {
		return fmt.Errorf("gas consumed should be %d, but got %d", gasConsumed, gasUsed)
	}

	return nil
}
