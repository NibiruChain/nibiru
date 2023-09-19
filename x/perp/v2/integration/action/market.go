package action

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// Logger
type logger struct {
	log string
}

func (e logger) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	fmt.Println(e.log)
	return ctx, nil, true
}

func Log(log string) action.Action {
	return logger{
		log: log,
	}
}

// createMarketAction creates a market
type createMarketAction struct {
	Market types.Market
	AMM    types.AMM
}

func (c createMarketAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	app.PerpKeeperV2.MarketLastVersion.Insert(ctx, c.Market.Pair, types.MarketLastVersion{Version: 1})
	app.PerpKeeperV2.SaveMarket(ctx, c.Market)
	app.PerpKeeperV2.SaveAMM(ctx, c.AMM)

	app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(c.AMM.Pair, ctx.BlockTime()), types.ReserveSnapshot{
		Amm:         c.AMM,
		TimestampMs: ctx.BlockTime().UnixMilli(),
	})

	return ctx, nil, true
}

// CreateCustomMarket creates a market with custom parameters
func CreateCustomMarket(pair asset.Pair, marketModifiers ...MarketModifier) action.Action {
	market := types.DefaultMarket(pair)
	amm := types.AMM{
		Pair:            pair,
		Version:         1,
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

type MarketModifier func(market *types.Market, amm *types.AMM)

func WithPrepaidBadDebt(amount sdkmath.Int) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		market.PrepaidBadDebt = sdk.NewCoin(market.Pair.QuoteDenom(), amount)
	}
}

func WithPricePeg(newValue sdk.Dec) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		amm.PriceMultiplier = newValue
	}
}

func WithTotalLong(amount sdk.Dec) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		amm.TotalLong = amount
	}
}

func WithTotalShort(amount sdk.Dec) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		amm.TotalShort = amount
	}
}

func WithSqrtDepth(amount sdk.Dec) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		amm.SqrtDepth = amount
		amm.BaseReserve = amount
		amm.QuoteReserve = amount
	}
}

func WithLatestMarketCPF(amount sdk.Dec) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		market.LatestCumulativePremiumFraction = amount
	}
}

func WithMaxFundingRate(amount sdk.Dec) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		market.MaxFundingRate = amount
	}
}

func WithVersion(version uint64) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		market.Version = version
		amm.Version = version
	}
}

func WithEnabled(enabled bool) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		market.Enabled = enabled
	}
}

type editPriceMultiplier struct {
	pair     asset.Pair
	newValue sdk.Dec
}

func (e editPriceMultiplier) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.EditPriceMultiplier(ctx, e.pair, e.newValue)
	return ctx, err, true
}

func EditPriceMultiplier(pair asset.Pair, newValue sdk.Dec) action.Action {
	return editPriceMultiplier{
		pair:     pair,
		newValue: newValue,
	}
}

type editSwapInvariant struct {
	pair     asset.Pair
	newValue sdk.Dec
}

func (e editSwapInvariant) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.EditSwapInvariant(ctx, e.pair, e.newValue)
	return ctx, err, true
}

func EditSwapInvariant(pair asset.Pair, newValue sdk.Dec) action.Action {
	return editSwapInvariant{
		pair:     pair,
		newValue: newValue,
	}
}

type createPool struct {
	pair   asset.Pair
	market types.Market
	amm    types.AMM
}

func (c createPool) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.Admin().CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            c.pair,
		PriceMultiplier: c.amm.PriceMultiplier,
		SqrtDepth:       c.amm.SqrtDepth,
		Market:          &c.market,
	})
	return ctx, err, true
}

func CreateMarket(pair asset.Pair, market types.Market, amm types.AMM) action.Action {
	return createPool{
		pair:   pair,
		market: market,
		amm:    amm,
	}
}
