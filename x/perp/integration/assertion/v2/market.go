package assertion

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func MarketShouldBeEqual(market v2types.Market) marketShouldBeEqual {
	return marketShouldBeEqual{
		ExpectedMarket: market,
	}
}

type marketShouldBeEqual struct {
	ExpectedMarket v2types.Market
}

func (m marketShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	market, err := app.PerpKeeperV2.Markets.Get(ctx, m.ExpectedMarket.Pair)
	if err != nil {
		return ctx, err, false
	}

	if err := v2types.MarketsAreEqual(&m.ExpectedMarket, &market); err != nil {
		return ctx, err, false
	}

	return ctx, nil, false
}

type marketLatestCPFEqual struct {
	Pair                                    asset.Pair
	ExpectedLatestCumulativePremiumFraction sdk.Dec
}

func (m marketLatestCPFEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	market, err := app.PerpKeeperV2.Markets.Get(ctx, m.Pair)
	if err != nil {
		return ctx, err, false
	}

	if !market.LatestCumulativePremiumFraction.Equal(m.ExpectedLatestCumulativePremiumFraction) {
		return ctx, err, false
	}

	return ctx, nil, false
}

func MarketLatestCPFShouldBeEqual(pair asset.Pair, expectedCPF sdk.Dec) marketLatestCPFEqual {
	return marketLatestCPFEqual{
		Pair:                                    pair,
		ExpectedLatestCumulativePremiumFraction: expectedCPF,
	}
}
