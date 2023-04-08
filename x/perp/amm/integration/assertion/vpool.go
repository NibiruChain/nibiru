package assertion

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

type MarketChecker func(resp types.Market) error

type vpoolShouldBeEqual struct {
	pair     asset.Pair
	checkers []MarketChecker
}

func VpoolShouldBeEqual(pair asset.Pair, checkers ...MarketChecker) action.Action {
	return &vpoolShouldBeEqual{pair: pair, checkers: checkers}
}

func (v vpoolShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	vpool, err := app.VpoolKeeper.GetPool(ctx, v.pair)
	if err != nil {
		return ctx, err
	}

	for _, checker := range v.checkers {
		if err := checker(vpool); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

// MarketCheckers

// VPool_BiasShouldBeEqualTo checks if the bias is equal to the expected bias
func VPool_BiasShouldBeEqualTo(bias sdk.Dec) MarketChecker {
	return func(vpool types.Market) error {
		if vpool.Bias.Equal(bias) {
			return nil
		}
		return fmt.Errorf("expected bias %s, got %s", bias, vpool.Bias)
	}
}
