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
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

// CreateMarketAction creates a market
type CreateMarketAction struct {
	Market v2types.Market
	AMM    v2types.AMM
}

func (c CreateMarketAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
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

	return CreateMarketAction{
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
