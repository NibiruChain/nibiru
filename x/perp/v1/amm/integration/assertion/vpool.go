package assertion

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
)

type MarketChecker func(resp types.Market) error

type marketShouldBeEqual struct {
	pair     asset.Pair
	checkers []MarketChecker
}

func MarketShouldBeEqual(pair asset.Pair, checkers ...MarketChecker) action.Action {
	return &marketShouldBeEqual{pair: pair, checkers: checkers}
}

func (v marketShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	market, err := app.PerpAmmKeeper.GetPool(ctx, v.pair)
	if err != nil {
		return ctx, err, false
	}

	for _, checker := range v.checkers {
		if err := checker(market); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}

// MarketCheckers

// Market_BiasShouldBeEqualTo checks if the bias is equal to the expected bias
func Market_BiasShouldBeEqualTo(bias sdk.Dec) MarketChecker {
	return func(market types.Market) error {
		if market.GetBias().Equal(bias) {
			return nil
		}
		return fmt.Errorf("expected bias %s, got %s", bias, market.GetBias())
	}
}
