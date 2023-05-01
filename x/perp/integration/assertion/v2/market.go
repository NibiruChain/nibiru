package assertion

import (
	"fmt"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type MarketChecker func(resp v2types.Market) error

type marketShouldBeEqual struct {
	Pair     asset.Pair
	Checkers []MarketChecker
}

func (m marketShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	market, err := app.PerpKeeperV2.Markets.Get(ctx, m.Pair)
	if err != nil {
		return ctx, err, false
	}

	for _, checker := range m.Checkers {
		if err := checker(market); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}

func MarketShouldBeEqual(pair asset.Pair, marketCheckers ...MarketChecker) marketShouldBeEqual {
	return marketShouldBeEqual{
		Pair:     pair,
		Checkers: marketCheckers,
	}
}

func MarketLatestCPFShouldBeEqualTo(expectedCPF sdk.Dec) MarketChecker {
	return func(market v2types.Market) error {
		if !market.LatestCumulativePremiumFraction.Equal(expectedCPF) {
			return fmt.Errorf("expected latest cumulative premium fraction to be %s, got %s", expectedCPF, market.LatestCumulativePremiumFraction)
		}
		return nil
	}
}

func MarketPrepaidBadDebtShouldBeEqualTo(expectedAmount sdk.Int) MarketChecker {
	return func(market v2types.Market) error {
		expectedBadDebt := sdk.NewCoin(market.Pair.QuoteDenom(), expectedAmount)
		if !market.PrepaidBadDebt.Equal(expectedBadDebt) {
			return fmt.Errorf("expected prepaid bad debt to be %s, got %s", expectedBadDebt, market.PrepaidBadDebt)
		}
		return nil
	}
}
