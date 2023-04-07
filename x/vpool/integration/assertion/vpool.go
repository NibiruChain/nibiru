package assertion

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

type VpoolChecker func(resp types.Vpool) error

type vpoolShouldBeEqual struct {
	pair     asset.Pair
	checkers []VpoolChecker
}

func VpoolShouldBeEqual(pair asset.Pair, checkers ...VpoolChecker) action.Action {
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

// VpoolCheckers

// VPool_BiasShouldBeEqualTo checks if the bias is equal to the expected bias
func VPool_BiasShouldBeEqualTo(bias sdk.Dec) VpoolChecker {
	return func(vpool types.Vpool) error {
		if vpool.Bias.Equal(bias) {
			return nil
		}
		return fmt.Errorf("expected bias %s, got %s", bias, vpool.Bias)
	}
}
