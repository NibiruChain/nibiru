package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/v1/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v1/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
)

// CreateMarketAction creates a market
type CreateMarketAction struct {
	Pair asset.Pair

	QuoteReserve sdk.Dec
	BaseReserve  sdk.Dec

	MarketConfig perpammtypes.MarketConfig
}

func (c CreateMarketAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpAmmKeeper.CreatePool(
		ctx,
		c.Pair,
		c.QuoteReserve,
		c.BaseReserve,
		c.MarketConfig,
		sdk.OneDec(),
	)
	if err != nil {
		return ctx, err, true
	}

	keeper.SetPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
		Pair:                            c.Pair,
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
	})

	return ctx, nil, true
}

// CreateBaseMarket creates a base market with:
// - pair: ubtc:uusdc
// - quote asset reserve: 1000
// - base asset reserve: 100
// - market config: default
func CreateBaseMarket() CreateMarketAction {
	return CreateMarketAction{
		Pair:         asset.NewPair(denoms.BTC, denoms.USDC),
		QuoteReserve: sdk.NewDec(1000e6),
		BaseReserve:  sdk.NewDec(100e6),
		MarketConfig: perpammtypes.MarketConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("1"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("1"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.NewDec(10),
		},
	}
}

// CreateCustomMarket creates a market with custom parameters
func CreateCustomMarket(
	pair asset.Pair,
	quoteReserve, baseReserve sdk.Dec,
	marketConfig perpammtypes.MarketConfig,
) action.Action {
	return CreateMarketAction{
		Pair:         pair,
		QuoteReserve: quoteReserve,
		BaseReserve:  baseReserve,
		MarketConfig: marketConfig,
	}
}
