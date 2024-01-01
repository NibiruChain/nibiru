package action

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// createMarketAction creates a market
type createMarketAction struct {
	Market types.Market
	AMM    types.AMM
}

func (c createMarketAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	app.PerpKeeperV2.MarketLastVersion.Insert(
		ctx, c.Market.Pair, types.MarketLastVersion{Version: c.Market.Version})
	app.PerpKeeperV2.SaveMarket(ctx, c.Market)
	app.PerpKeeperV2.SaveAMM(ctx, c.AMM)

	app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(c.AMM.Pair, ctx.BlockTime()), types.ReserveSnapshot{
		Amm:         c.AMM,
		TimestampMs: ctx.BlockTime().UnixMilli(),
	})

	app.PerpKeeperV2.Collateral.Set(ctx, types.TestingCollateralDenomNUSD)

	return ctx, nil
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

func WithPrepaidBadDebt(amount sdkmath.Int, collateral string) MarketModifier {
	return func(market *types.Market, amm *types.AMM) {
		market.PrepaidBadDebt = sdk.NewCoin(collateral, amount)
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

type shiftPegMultiplier struct {
	pair     asset.Pair
	newValue sdk.Dec
}

func (e shiftPegMultiplier) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.PerpKeeperV2.Sudo().ShiftPegMultiplier(
		ctx, e.pair, e.newValue, testapp.DefaultSudoRoot(),
	)
	return ctx, err
}

func ShiftPegMultiplier(pair asset.Pair, newValue sdk.Dec) action.Action {
	return shiftPegMultiplier{
		pair:     pair,
		newValue: newValue,
	}
}

type shiftSwapInvariant struct {
	pair     asset.Pair
	newValue sdkmath.Int
}

func (e shiftSwapInvariant) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.PerpKeeperV2.Sudo().ShiftSwapInvariant(
		ctx, e.pair, e.newValue, testapp.DefaultSudoRoot(),
	)
	return ctx, err
}

func ShiftSwapInvariant(pair asset.Pair, newValue sdkmath.Int) action.Action {
	return shiftSwapInvariant{
		pair:     pair,
		newValue: newValue,
	}
}

type createPool struct {
	pair   asset.Pair
	market types.Market
	amm    types.AMM
}

func (c createPool) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.PerpKeeperV2.Sudo().CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            c.pair,
		PriceMultiplier: c.amm.PriceMultiplier,
		SqrtDepth:       c.amm.SqrtDepth,
		Market:          &c.market,
	})
	return ctx, err
}

func CreateMarket(pair asset.Pair, market types.Market, amm types.AMM) action.Action {
	return createPool{
		pair:   pair,
		market: market,
		amm:    amm,
	}
}

type setCollateral struct {
	Denom  string
	Sender string
}

func (c setCollateral) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	sudoers, err := app.SudoKeeper.Sudoers.Get(ctx)
	if err != nil {
		return ctx, err
	}
	sudoers.Root = common.NIBIRU_TEAM
	app.SudoKeeper.Sudoers.Set(ctx, sudoers)

	senderAddr, err := sdk.AccAddressFromBech32(c.Sender)
	if err != nil {
		return ctx, err
	}
	err = app.PerpKeeperV2.Sudo().ChangeCollateralDenom(ctx, c.Denom, senderAddr)
	return ctx, err
}

func SetCollateral(denom string) action.Action {
	return setCollateral{
		Denom:  denom,
		Sender: common.NIBIRU_TEAM,
	}
}
