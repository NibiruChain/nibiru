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

func (p positionNotionalTwapShouldBeEqualTo) IsNotMandatory() {}

func (p positionNotionalTwapShouldBeEqualTo) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	position, err := app.PerpKeeperV2.GetPosition(ctx, p.pair, 1, p.trader)
	if err != nil {
		return ctx, err
	}

	notionalValue, err := app.PerpKeeperV2.PositionNotionalTWAP(ctx, position, p.twapLookbackWindow)
	if err != nil {
		return ctx, err
	}

	if !notionalValue.Equal(p.expectedNotionalValue) {
		return ctx, fmt.Errorf("notional value expected to be %s, received %s", p.expectedNotionalValue, notionalValue)
	}

	return ctx, nil
}

func PositionNotionalTWAPShouldBeEqualTo(pair asset.Pair, trader sdk.AccAddress, twapLookbackWindow time.Duration, expectedNotionalValue sdk.Dec) action.Action {
	return positionNotionalTwapShouldBeEqualTo{
		pair:                  pair,
		trader:                trader,
		twapLookbackWindow:    twapLookbackWindow,
		expectedNotionalValue: expectedNotionalValue,
	}
}
