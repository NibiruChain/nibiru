package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
)

func TestPoolHasEnoughQuoteReserve(t *testing.T) {
	pair, err := common.NewAssetPair("BTC:NUSD")
	require.NoError(t, err)

	pool := NewPool(
		pair,
		sdk.MustNewDecFromStr("0.9"), // 0.9
		sdk.NewDec(10_000_000),       // 10
		sdk.NewDec(10_000_000),       // 10
		sdk.MustNewDecFromStr("0.1"),
		sdk.MustNewDecFromStr("0.1"),
	)

	// less that max ratio
	require.True(t, pool.HasEnoughQuoteReserve(sdk.NewDec(8_000_000)))

	// equal to ratio limit
	require.True(t, pool.HasEnoughQuoteReserve(sdk.NewDec(9_000_000)))

	// more than ratio limit
	require.False(t, pool.HasEnoughQuoteReserve(sdk.NewDec(9_000_001)))
}

func TestGetBaseAmountByQuoteAmount(t *testing.T) {
	pair, err := common.NewAssetPair("BTC:NUSD")
	require.NoError(t, err)

	tests := []struct {
		name               string
		baseAssetReserve   sdk.Dec
		quoteAssetReserve  sdk.Dec
		quoteAmount        sdk.Dec
		direction          Direction
		expectedBaseAmount sdk.Dec
		expectedErr        error
	}{
		{
			name:               "quote amount zero",
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.ZeroDec(),
			direction:          Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.ZeroDec(),
		},
		{
			name:               "simple add quote to pool",
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.MustNewDecFromStr("333.333333333333333333"),
		},
		{
			name:               "simple remove quote from pool",
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          Direction_REMOVE_FROM_POOL,
			expectedBaseAmount: sdk.NewDec(1000),
		},
		{
			name:              "too much quote removed results in error",
			baseAssetReserve:  sdk.NewDec(1000),
			quoteAssetReserve: sdk.NewDec(1000),
			quoteAmount:       sdk.NewDec(1000),
			direction:         Direction_REMOVE_FROM_POOL,
			expectedErr:       ErrQuoteReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := NewPool(
				/*pair=*/ pair,
				/*tradeLimitRatio=*/ sdk.MustNewDecFromStr("0.9"),
				/*quoteAssetReserve=*/ tc.quoteAssetReserve,
				/*baseAssetReserve=*/ tc.baseAssetReserve,
				/*fluctuationLimitRatio=*/ sdk.MustNewDecFromStr("0.1"),
				/*maxOracleSpreadRatio=*/ sdk.MustNewDecFromStr("0.1"),
			)

			amount, err := pool.GetBaseAmountByQuoteAmount(tc.direction, tc.quoteAmount)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr,
					"expected error: %w, got: %w", tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.EqualValuesf(t, tc.expectedBaseAmount, amount,
					"expected quote: %s, got: %s", tc.expectedBaseAmount.String(), amount.String(),
				)
			}
		})
	}
}

func TestGetQuoteAmountByBaseAmount(t *testing.T) {
	pair, err := common.NewAssetPair("BTC:NUSD")
	require.NoError(t, err)

	tests := []struct {
		name                string
		baseAssetReserve    sdk.Dec
		quoteAssetReserve   sdk.Dec
		baseAmount          sdk.Dec
		direction           Direction
		expectedQuoteAmount sdk.Dec
		expectedErr         error
	}{
		{
			name:                "base amount zero",
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.ZeroDec(),
			direction:           Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.ZeroDec(),
		},
		{
			name:                "simple add base to pool",
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.NewDec(500),
			direction:           Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.MustNewDecFromStr("333.333333333333333333"),
		},
		{
			name:                "simple remove base from pool",
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.NewDec(500),
			direction:           Direction_REMOVE_FROM_POOL,
			expectedQuoteAmount: sdk.NewDec(1000),
		},
		{
			name:              "too much base removed results in error",
			baseAssetReserve:  sdk.NewDec(1000),
			quoteAssetReserve: sdk.NewDec(1000),
			baseAmount:        sdk.NewDec(1000),
			direction:         Direction_REMOVE_FROM_POOL,
			expectedErr:       ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := NewPool(
				/*pair=*/ pair,
				/*tradeLimitRatio=*/ sdk.OneDec(),
				/*quoteAssetReserve=*/ tc.quoteAssetReserve,
				/*baseAssetReserve=*/ tc.baseAssetReserve,
				/*fluctuationLimitRatio=*/ sdk.OneDec(),
				/*maxOracleSpreadRatio=*/ sdk.OneDec(),
			)

			amount, err := pool.GetQuoteAmountByBaseAmount(tc.direction, tc.baseAmount)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr,
					"expected error: %w, got: %w", tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.EqualValuesf(t, tc.expectedQuoteAmount, amount,
					"expected quote: %s, got: %s", tc.expectedQuoteAmount.String(), amount.String(),
				)
			}
		})
	}
}

func TestIncreaseDecreaseReserves(t *testing.T) {
	pair, err := common.NewAssetPair("ATOM:NUSD")
	require.NoError(t, err)

	pool := NewPool(
		pair,
		/*tradeLimitRatio=*/ sdk.MustNewDecFromStr("0.9"),
		/*quoteAssetReserve=*/ sdk.NewDec(1_000_000),
		/*baseAssetReserve*/ sdk.NewDec(1_000_000),
		/*fluctuationLimitRatio*/ sdk.MustNewDecFromStr("0.1"),
		/*maxOracleSpreadRatio*/ sdk.MustNewDecFromStr("0.01"),
	)

	t.Log("decrease quote asset reserve")
	pool.DecreaseQuoteAssetReserve(sdk.NewDec(100))
	require.Equal(t, sdk.NewDec(999_900), pool.QuoteAssetReserve)

	t.Log("increase quote asset reserve")
	pool.IncreaseQuoteAssetReserve(sdk.NewDec(100))
	require.Equal(t, sdk.NewDec(1_000_000), pool.QuoteAssetReserve)

	t.Log("decrease base asset reserve")
	pool.DecreaseBaseAssetReserve(sdk.NewDec(100))
	require.Equal(t, sdk.NewDec(999_900), pool.BaseAssetReserve)

	t.Log("incrase base asset reserve")
	pool.IncreaseBaseAssetReserve(sdk.NewDec(100))
	require.Equal(t, sdk.NewDec(1_000_000), pool.BaseAssetReserve)
}
