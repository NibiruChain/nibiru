package assertion

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type positionNotionalTwapShouldBeEqualTo struct {
	pair                  asset.Pair
	trader                sdk.AccAddress
	twapLookbackWindow    time.Duration
	expectedNotionalValue sdk.Dec
}

func (p positionNotionalTwapShouldBeEqualTo) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	position, err := app.PerpKeeperV2.GetPosition(ctx, p.pair, 1, p.trader)
	if err != nil {
		return ctx, err, false
	}

	notionalValue, err := app.PerpKeeperV2.PositionNotionalTWAP(ctx, position, p.twapLookbackWindow)
	if err != nil {
		return ctx, err, false
	}

	if !notionalValue.Equal(p.expectedNotionalValue) {
		return ctx, fmt.Errorf("notional value expected to be %s, received %s", p.expectedNotionalValue, notionalValue), false
	}

	return ctx, nil, false
}

func PositionNotionalTWAPShouldBeEqualTo(pair asset.Pair, trader sdk.AccAddress, twapLookbackWindow time.Duration, expectedNotionalValue sdk.Dec) action.Action {
	return positionNotionalTwapShouldBeEqualTo{
		pair:                  pair,
		trader:                trader,
		twapLookbackWindow:    twapLookbackWindow,
		expectedNotionalValue: expectedNotionalValue,
	}
}
