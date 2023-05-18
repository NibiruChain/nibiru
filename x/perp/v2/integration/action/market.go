package action

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"

	"github.com/NibiruChain/nibiru/app"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// createMarketAction creates a market
type createMarketAction struct {
	Market v2types.Market
	AMM    v2types.AMM
}

func (c createMarketAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	app.PerpKeeperV2.Markets.Insert(ctx, c.Market.Pair, c.Market)
	app.PerpKeeperV2.AMMs.Insert(ctx, c.AMM.Pair, c.AMM)

	app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(c.AMM.Pair, ctx.BlockTime()), v2types.ReserveSnapshot{
		Amm:         c.AMM,
		TimestampMs: ctx.BlockTime().UnixMilli(),
	})

	return ctx, nil, true
}

// CreateCustomMarket creates a market with custom parameters
func CreateCustomMarket(pair asset.Pair, marketModifiers ...marketModifier) action.Action {
	market := v2types.Market{
		Pair:                            pair,
		Enabled:                         true,
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
		EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
		LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
		PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5000"),
		FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
		TwapLookbackWindow:              time.Minute * 30,
		PrepaidBadDebt:                  sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
		PriceFluctuationLimitRatio:      sdk.MustNewDecFromStr("0.1000"),
		MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.0625"),
		MaxLeverage:                     sdk.NewDec(10),
	}

	amm := v2types.AMM{
		Pair:            pair,
		BaseReserve:     sdk.NewDec(1e12),
		QuoteReserve:    sdk.NewDec(1e12),
		SqrtDepth:       sdk.NewDec(1e12),
		PriceMultiplier: sdk.OneDec(),
		TotalLong:       sdk.ZeroDec(),
		TotalShort:      sdk.ZeroDec(),
	}

	for _, modifier := range marketModifiers {
		modifier(&market, &amm)
	}

	return createMarketAction{
		Market: market,
		AMM:    amm,
	}
}

type marketModifier func(market *v2types.Market, amm *v2types.AMM)

func WithPrepaidBadDebt(amount sdk.Int) marketModifier {
	return func(market *v2types.Market, amm *v2types.AMM) {
		market.PrepaidBadDebt = sdk.NewCoin(market.Pair.QuoteDenom(), amount)
	}
}

func WithPricePeg(multiplier sdk.Dec) marketModifier {
	return func(market *v2types.Market, amm *v2types.AMM) {
		amm.PriceMultiplier = multiplier
	}
}

func WithTotalLong(amount sdk.Dec) marketModifier {
	return func(market *v2types.Market, amm *v2types.AMM) {
		amm.TotalLong = amount
	}
}

func WithTotalShort(amount sdk.Dec) marketModifier {
	return func(market *v2types.Market, amm *v2types.AMM) {
		amm.TotalShort = amount
	}
}

func WithSqrtDepth(amount sdk.Dec) marketModifier {
	return func(market *v2types.Market, amm *v2types.AMM) {
		amm.SqrtDepth = amount
		amm.BaseReserve = amount
		amm.QuoteReserve = amount
	}
}

type editPriceMultiplier struct {
	pair       asset.Pair
	multiplier sdk.Dec
}

func (e editPriceMultiplier) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.EditPriceMultiplier(ctx, e.pair, e.multiplier)
	return ctx, err, true
}

func EditPriceMultiplier(pair asset.Pair, multiplier sdk.Dec) action.Action {
	return editPriceMultiplier{
		pair:       pair,
		multiplier: multiplier,
	}
}

type editSwapInvariant struct {
	pair       asset.Pair
	multiplier sdk.Dec
}

func (e editSwapInvariant) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.EditSwapInvariant(ctx, e.pair, e.multiplier)
	return ctx, err, true
}

func EditSwapInvariant(pair asset.Pair, multiplier sdk.Dec) action.Action {
	return editSwapInvariant{
		pair:       pair,
		multiplier: multiplier,
	}
}

type createPool struct {
	pair   asset.Pair
	market v2types.Market
	amm    v2types.AMM
}

func (c createPool) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.CreatePool(ctx, c.pair, c.market, c.amm)
	return ctx, err, true
}

func CreatePool(pair asset.Pair, market v2types.Market, amm v2types.AMM) action.Action {
	return createPool{
		pair:   pair,
		market: market,
		amm:    amm,
	}
}
