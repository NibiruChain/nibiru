package types

import (
	"testing"

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
	market := new(Market)

	// Test when all values are within expected ranges
	market.WithMaintenanceMarginRatio(sdk.NewDecWithPrec(1, 1))
	market.WithEcosystemFee(sdk.NewDecWithPrec(3, 1))
	market.WithExchangeFee(sdk.NewDecWithPrec(4, 1))
	market.WithLiquidationFee(sdk.NewDecWithPrec(2, 1))
	market.WithPartialLiquidationRatio(sdk.NewDecWithPrec(1, 1))
	market.WithMaxLeverage(sdk.NewDec(10))
	require.NoError(t, market.Validate())

	testCases := []struct {
		modifier      func(*Market)
		requiredError string
	}{
		{
			modifier:      func(m *Market) { m.WithMaintenanceMarginRatio(sdk.NewDec(-1)) },
			requiredError: "maintenance margin ratio ratio must be 0 <= ratio <= 1"},
		{
			modifier:      func(m *Market) { m.WithEcosystemFee(sdk.NewDec(2)) },
			requiredError: "ecosystem fund fee ratio must be 0 <= ratio <= 1"},
		{
			modifier:      func(m *Market) { m.WithExchangeFee(sdk.NewDec(-1)) },
			requiredError: "exchange fee ratio must be 0 <= ratio <= 1"},
		{
			modifier:      func(m *Market) { m.WithLiquidationFee(sdk.NewDec(2)) },
			requiredError: "liquidation fee ratio must be 0 <= ratio <= 1"},
		{
			modifier:      func(m *Market) { m.WithPartialLiquidationRatio(sdk.NewDec(-1)) },
			requiredError: "partial liquidation ratio must be 0 <= ratio <= 1"},
		{
			modifier:      func(m *Market) { m.WithMaxLeverage(sdk.ZeroDec()) },
			requiredError: "max leverage must be > 0"},
		{
			modifier:      func(m *Market) { m.WithMaxLeverage(sdk.NewDec(20)).WithMaintenanceMarginRatio(sdk.NewDec(1)) },
			requiredError: "margin ratio opened with max leverage position will be lower than Maintenance margin ratio"},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.requiredError, func(t *testing.T) {
			newMarket := market.copy()

			tc.modifier(newMarket)

			err := newMarket.Validate()
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
	market2.WithMaintenanceMarginRatio(sdk.NewDecWithPrec(3, 1))
	require.Error(t, MarketsAreEqual(market1, market2))

	require.NoError(t, MarketsAreEqual(market1, market1.copy()))
}
