package assertion

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
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

func Market_LatestCPFShouldBeEqualTo(expectedCPF sdk.Dec) MarketChecker {
	return func(market v2types.Market) error {
		if !market.LatestCumulativePremiumFraction.Equal(expectedCPF) {
			return fmt.Errorf("expected latest cumulative premium fraction to be %s, got %s", expectedCPF, market.LatestCumulativePremiumFraction)
		}
		return nil
	}
}

func Market_PrepaidBadDebtShouldBeEqualTo(expectedAmount sdk.Int) MarketChecker {
	return func(market v2types.Market) error {
		expectedBadDebt := sdk.NewCoin(market.Pair.QuoteDenom(), expectedAmount)
		if !market.PrepaidBadDebt.Equal(expectedBadDebt) {
			return fmt.Errorf("expected prepaid bad debt to be %s, got %s", expectedBadDebt, market.PrepaidBadDebt)
		}
		return nil
	}
}

type ammShouldBeEqual struct {
	Pair     asset.Pair
	Checkers []AMMChecker
}

type AMMChecker func(amm v2types.AMM) error

func (a ammShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	amm, err := app.PerpKeeperV2.AMMs.Get(ctx, a.Pair)
	if err != nil {
		return ctx, err, false
	}

	for _, checker := range a.Checkers {
		if err := checker(amm); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}

func AMMShouldBeEqual(pair asset.Pair, ammCheckers ...AMMChecker) ammShouldBeEqual {
	return ammShouldBeEqual{
		Pair:     pair,
		Checkers: ammCheckers,
	}
}

func AMM_BaseReserveShouldBeEqual(expectedBaseReserve sdk.Dec) AMMChecker {
	return func(amm v2types.AMM) error {
		if !amm.BaseReserve.Equal(expectedBaseReserve) {
			return fmt.Errorf("expected base reserve to be %s, got %s", expectedBaseReserve, amm.BaseReserve)
		}
		return nil
	}
}

func AMM_QuoteReserveShouldBeEqual(expectedQuoteReserve sdk.Dec) AMMChecker {
	return func(amm v2types.AMM) error {
		if !amm.QuoteReserve.Equal(expectedQuoteReserve) {
			return fmt.Errorf("expected quote reserve to be %s, got %s", expectedQuoteReserve, amm.QuoteReserve)
		}
		return nil
	}
}

func AMM_SqrtDepthShouldBeEqual(expectedSqrtDepth sdk.Dec) AMMChecker {
	return func(amm v2types.AMM) error {
		if !amm.SqrtDepth.Equal(expectedSqrtDepth) {
			return fmt.Errorf("expected sqrt depth to be %s, got %s", expectedSqrtDepth, amm.SqrtDepth)
		}
		return nil
	}
}

func AMM_PriceMultiplierShouldBeEqual(expectedPriceMultiplier sdk.Dec) AMMChecker {
	return func(amm v2types.AMM) error {
		if !amm.PriceMultiplier.Equal(expectedPriceMultiplier) {
			return fmt.Errorf("expected price multiplier to be %s, got %s", expectedPriceMultiplier, amm.PriceMultiplier)
		}
		return nil
	}
}
