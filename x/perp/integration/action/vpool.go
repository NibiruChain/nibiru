package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
)

// CreateVPoolAction creates a vpool
type CreateVPoolAction struct {
	Pair asset.Pair

	QuoteAssetReserve sdk.Dec
	BaseAssetReserve  sdk.Dec

	MarketConfig perpammtypes.MarketConfig

	Bias sdk.Dec
}

func (c CreateVPoolAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.PerpAmmKeeper.CreatePool(
		ctx,
		c.Pair,
		c.QuoteAssetReserve,
		c.BaseAssetReserve,
		c.MarketConfig,
		c.Bias,
		sdk.OneDec(),
	)
	if err != nil {
		return ctx, err
	}

	keeper.SetPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
		Pair:                            c.Pair,
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
	})

	return ctx, nil
}

// CreateBaseMarket creates a base vpool with:
// - pair: ubtc:uusdc
// - quote asset reserve: 1000
// - base asset reserve: 100
// - vpool config: default
func CreateBaseMarket() CreateVPoolAction {
	return CreateVPoolAction{
		Pair:              asset.NewPair(denoms.BTC, denoms.USDC),
		QuoteAssetReserve: sdk.NewDec(1000e6),
		BaseAssetReserve:  sdk.NewDec(100e6),
		MarketConfig: perpammtypes.MarketConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("1"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("1"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.NewDec(10),
		},
	}
}

// CreateCustomMarket creates a vpool with custom parameters
func CreateCustomMarket(
	pair asset.Pair,
	quoteAssetReserve, baseAssetReserve sdk.Dec,
	marketConfig perpammtypes.MarketConfig,
	bias sdk.Dec,
) action.Action {
	return CreateVPoolAction{
		Pair:              pair,
		QuoteAssetReserve: quoteAssetReserve,
		BaseAssetReserve:  baseAssetReserve,
		MarketConfig:      marketConfig,
		Bias:              bias,
	}
}
