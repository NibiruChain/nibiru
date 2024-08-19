package assertion

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/action"
)

type gasConsumedShouldBe struct {
	gasConsumed uint64
}

func (g gasConsumedShouldBe) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	gasUsed := ctx.GasMeter().GasConsumed()
	if g.gasConsumed != gasUsed {
		return ctx, fmt.Errorf("gas consumed should be %d, but got %d", g.gasConsumed, gasUsed)
	}

	return ctx, nil
}

func GasConsumedShouldBe(gasConsumed uint64) action.Action {
	return &gasConsumedShouldBe{gasConsumed: gasConsumed}
}
