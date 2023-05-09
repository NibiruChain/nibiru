package assertion

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type twalShouldBe struct {
	pair               asset.Pair
	twapCalcOpt        v2types.TwapCalcOption
	dir                v2types.Direction
	assetAmt           sdk.Dec
	twapLookbackWindow time.Duration

	expectedTwap sdk.Dec
}

func (c twalShouldBe) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	twap, err := app.PerpKeeperV2.CalcTwap(ctx, c.pair, c.twapCalcOpt, c.dir, c.assetAmt, c.twapLookbackWindow)
	if err != nil {
		return ctx, err, false
	}

	if !twap.Equal(c.expectedTwap) {
		return ctx, fmt.Errorf("invalid twap, expected %s, received %s", c.expectedTwap, twap), false
	}

	return ctx, nil, false
}

func TwapShouldBe(pair asset.Pair, twapCalcOpt v2types.TwapCalcOption, dir v2types.Direction, assetAmt sdk.Dec, twapLookbackWindow time.Duration, expectedTwap sdk.Dec) action.Action {
	return twalShouldBe{
		pair:               pair,
		twapCalcOpt:        twapCalcOpt,
		dir:                dir,
		assetAmt:           assetAmt,
		twapLookbackWindow: twapLookbackWindow,
		expectedTwap:       expectedTwap,
	}
}
