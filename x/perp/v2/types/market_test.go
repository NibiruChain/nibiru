package types

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestIsPercent(t *testing.T) {
	require.True(t, isPercent(sdk.ZeroDec()))
	require.True(t, isPercent(sdk.NewDec(1)))
	require.True(t, isPercent(sdk.NewDecWithPrec(5, 1))) // 0.5
	require.False(t, isPercent(sdk.NewDec(-1)))
	require.False(t, isPercent(sdk.NewDec(2)))
}

func TestValidate(t *testing.T) {
	// Test when all values are within expected ranges
	market := Market{}.
		WithMaintenanceMarginRatio(sdk.NewDecWithPrec(1, 1)).
		WithEcosystemFee(sdk.NewDecWithPrec(3, 1)).
		WithExchangeFee(sdk.NewDecWithPrec(4, 1)).
		WithLiquidationFee(sdk.NewDecWithPrec(2, 1)).
		WithPartialLiquidationRatio(sdk.NewDecWithPrec(1, 1)).
		WithMaxLeverage(sdk.NewDec(10)).
		WithMaxFundingRate(sdk.NewDec(1))
	require.NoError(t, market.Validate())

	testCases := []struct {
		modifier      func(Market) Market
		requiredError string
	}{
		{
			modifier:      func(m Market) Market { return m.WithMaintenanceMarginRatio(sdk.NewDec(-1)) },
			requiredError: "maintenance margin ratio ratio must be 0 <= ratio <= 1",
		},
		{
			modifier:      func(m Market) Market { return m.WithEcosystemFee(sdk.NewDec(2)) },
			requiredError: "ecosystem fund fee ratio must be 0 <= ratio <= 1",
		},
		{
			modifier:      func(m Market) Market { return m.WithExchangeFee(sdk.NewDec(-1)) },
			requiredError: "exchange fee ratio must be 0 <= ratio <= 1",
		},
		{
			modifier:      func(m Market) Market { return m.WithLiquidationFee(sdk.NewDec(2)) },
			requiredError: "liquidation fee ratio must be 0 <= ratio <= 1",
		},
		{
			modifier:      func(m Market) Market { return m.WithPartialLiquidationRatio(sdk.NewDec(-1)) },
			requiredError: "partial liquidation ratio must be 0 <= ratio <= 1",
		},
		{
			modifier:      func(m Market) Market { return m.WithMaxLeverage(sdk.ZeroDec()) },
			requiredError: "max leverage must be > 0",
		},
		{
			modifier:      func(m Market) Market { return m.WithMaxFundingRate(sdk.NewDec(-1)) },
			requiredError: "max funding rate must be >= 0",
		},
		{
			modifier: func(m Market) Market {
				return m.WithMaxLeverage(sdk.NewDec(20)).WithMaintenanceMarginRatio(sdk.NewDec(1))
			},
			requiredError: "margin ratio opened with max leverage position will be lower than Maintenance margin ratio",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.requiredError, func(t *testing.T) {
			newMarket := tc.modifier(market)

			err := newMarket.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.requiredError)
		})
	}
}

func TestMarketEqual(t *testing.T) {
	market := Market{}.
		WithMaintenanceMarginRatio(sdk.NewDecWithPrec(1, 1)).
		WithEcosystemFee(sdk.NewDecWithPrec(3, 1)).
		WithExchangeFee(sdk.NewDecWithPrec(4, 1)).
		WithLiquidationFee(sdk.NewDecWithPrec(2, 1)).
		WithPartialLiquidationRatio(sdk.NewDecWithPrec(1, 1)).
		WithMaxLeverage(sdk.NewDec(10)).
		WithLatestCumulativePremiumFraction(sdk.OneDec()).
		WithMaxFundingRate(sdk.OneDec())

	// Test when all values are within expected ranges
	require.NoError(t, market.Validate())

	testCases := []struct {
		modifier      func(Market) Market
		requiredError string
	}{
		{
			modifier:      func(m Market) Market { return m.WithPair("ueth:unusd") },
			requiredError: "expected market pair",
		},
		{
			modifier:      func(m Market) Market { return m.WithMaintenanceMarginRatio(sdk.NewDec(42)) },
			requiredError: "expected market maintenance margin ratio",
		},
		{
			modifier:      func(m Market) Market { return m.WithMaxLeverage(sdk.NewDec(42)) },
			requiredError: "expected market max leverage",
		},
		{
			modifier:      func(m Market) Market { return m.WithPartialLiquidationRatio(sdk.NewDec(42)) },
			requiredError: "expected market partial liquidation ratio",
		},
		{
			modifier:      func(m Market) Market { return m.WithFundingRateEpochId("hi") },
			requiredError: "expected market funding rate epoch id",
		},
		{
			modifier:      func(m Market) Market { return m.WithMaxFundingRate(sdk.NewDec(42)) },
			requiredError: "expected market max funding rate",
		},
		{
			modifier:      func(m Market) Market { return m.WithLatestCumulativePremiumFraction(sdk.NewDec(42)) },
			requiredError: "expected market latest cumulative premium fraction",
		},
		{
			modifier:      func(m Market) Market { return m.WithEcosystemFundFeeRatio(sdk.NewDec(42)) },
			requiredError: "expected market ecosystem fund fee ratio",
		},
		{
			modifier:      func(m Market) Market { return m.WithExchangeFeeRatio(sdk.NewDec(42)) },
			requiredError: "expected market exchange fee ratio",
		},
		{
			modifier:      func(m Market) Market { return m.WithLiquidationFeeRatio(sdk.NewDec(42)) },
			requiredError: "expected market liquidation fee ratio",
		},
		{
			modifier:      func(m Market) Market { return m.WithPrepaidBadDebt(sdk.NewCoin("ubtc", sdk.OneInt())) },
			requiredError: "expected market prepaid bad debt",
		},
		{
			modifier:      func(m Market) Market { return m.WithEnabled(true) },
			requiredError: "expected market enabled",
		},
		{
			modifier:      func(m Market) Market { return m.WithTwapLookbackWindow(time.Minute) },
			requiredError: "expected market twap lookback window",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.requiredError, func(t *testing.T) {
			newMarket := tc.modifier(market)

			err := MarketsAreEqual(market, newMarket)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.requiredError)
		})
	}
}

func TestMarketsAreEqual(t *testing.T) {
	market1 := new(Market).WithMaintenanceMarginRatio(sdk.NewDecWithPrec(5, 1))
	market2 := new(Market).WithMaintenanceMarginRatio(sdk.NewDecWithPrec(5, 1))

	// Test when markets are equal
	require.NoError(t, MarketsAreEqual(market1, market2))

	// Test when markets are not equal
	market2 = market2.WithMaintenanceMarginRatio(sdk.NewDecWithPrec(3, 1))
	require.Error(t, MarketsAreEqual(market1, market2))

	require.NoError(t, MarketsAreEqual(market1, market1))
}
