package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestPoolHasEnoughQuoteReserve(t *testing.T) {
	pool := NewPool(
		"BTC:NUSD",
		sdk.MustNewDecFromStr("0.9"), // 0.9
		sdk.NewInt(10_000_000),       // 10
		sdk.NewInt(10_000_000),       // 10
		sdk.MustNewDecFromStr("0.1"),
	)

	// less that max ratio
	require.True(t, pool.HasEnoughQuoteReserve(sdk.NewInt(8_000_000)))

	// equal to ratio limit
	require.True(t, pool.HasEnoughQuoteReserve(sdk.NewInt(9_000_000)))

	// more than ratio limit
	require.False(t, pool.HasEnoughQuoteReserve(sdk.NewInt(9_000_001)))
}

func TestGetBaseAmountByQuoteAmount(t *testing.T) {
	tests := []struct {
		name               string
		baseAssetReserve   sdk.Int
		quoteAssetReserve  sdk.Int
		quoteAmount        sdk.Int
		direction          Direction
		expectedBaseAmount sdk.Int
		expectedErr        error
	}{
		{
			name:               "quote amount zero",
			baseAssetReserve:   sdk.NewIntFromUint64(1000),
			quoteAssetReserve:  sdk.NewIntFromUint64(1000),
			quoteAmount:        sdk.NewIntFromUint64(0),
			direction:          Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.NewIntFromUint64(0),
		},
		{
			name:               "simple add quote to pool",
			baseAssetReserve:   sdk.NewIntFromUint64(1000),
			quoteAssetReserve:  sdk.NewIntFromUint64(1000),
			quoteAmount:        sdk.NewIntFromUint64(500),
			direction:          Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.NewIntFromUint64(333), // rounds down
		},
		{
			name:               "simple remove quote from pool",
			baseAssetReserve:   sdk.NewIntFromUint64(1000),
			quoteAssetReserve:  sdk.NewIntFromUint64(1000),
			quoteAmount:        sdk.NewIntFromUint64(500),
			direction:          Direction_REMOVE_FROM_POOL,
			expectedBaseAmount: sdk.NewIntFromUint64(1000),
		},
		{
			name:              "too much quote removed results in error",
			baseAssetReserve:  sdk.NewIntFromUint64(1000),
			quoteAssetReserve: sdk.NewIntFromUint64(1000),
			quoteAmount:       sdk.NewIntFromUint64(1000),
			direction:         Direction_REMOVE_FROM_POOL,
			expectedErr:       ErrQuoteReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := NewPool(
				/*pair=*/ "BTC:NUSD",
				/*tradeLimitRatio=*/ sdk.MustNewDecFromStr("0.9"),
				/*quoteAssetReserve=*/ tc.quoteAssetReserve,
				/*baseAssetReserve=*/ tc.baseAssetReserve,
				/*fluctuationLimitRatio=*/ sdk.MustNewDecFromStr("0.1"),
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
	tests := []struct {
		name                string
		baseAssetReserve    sdk.Int
		quoteAssetReserve   sdk.Int
		baseAmount          sdk.Int
		direction           Direction
		expectedQuoteAmount sdk.Int
		expectedErr         error
	}{
		{
			name:                "base amount zero",
			baseAssetReserve:    sdk.NewIntFromUint64(1000),
			quoteAssetReserve:   sdk.NewIntFromUint64(1000),
			baseAmount:          sdk.NewIntFromUint64(0),
			direction:           Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.NewIntFromUint64(0),
		},
		{
			name:                "simple add base to pool",
			baseAssetReserve:    sdk.NewIntFromUint64(1000),
			quoteAssetReserve:   sdk.NewIntFromUint64(1000),
			baseAmount:          sdk.NewIntFromUint64(500),
			direction:           Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.NewIntFromUint64(333), // rounds down
		},
		{
			name:                "simple remove base from pool",
			baseAssetReserve:    sdk.NewIntFromUint64(1000),
			quoteAssetReserve:   sdk.NewIntFromUint64(1000),
			baseAmount:          sdk.NewIntFromUint64(500),
			direction:           Direction_REMOVE_FROM_POOL,
			expectedQuoteAmount: sdk.NewIntFromUint64(1000),
		},
		{
			name:              "too much base removed results in error",
			baseAssetReserve:  sdk.NewIntFromUint64(1000),
			quoteAssetReserve: sdk.NewIntFromUint64(1000),
			baseAmount:        sdk.NewIntFromUint64(1000),
			direction:         Direction_REMOVE_FROM_POOL,
			expectedErr:       ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := NewPool(
				/*pair=*/ "BTC:NUSD",
				/*tradeLimitRatio=*/ sdk.MustNewDecFromStr("0.9"),
				/*quoteAssetReserve=*/ tc.quoteAssetReserve,
				/*baseAssetReserve=*/ tc.baseAssetReserve,
				/*fluctuationLimitRatio=*/ sdk.MustNewDecFromStr("0.1"),
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
	pool := NewPool(
		"ATOM:NUSD",
		sdk.MustNewDecFromStr("0.9"),
		sdk.NewInt(1_000_000),
		sdk.NewInt(1_000_000),
		sdk.MustNewDecFromStr("0.1"),
	)

	t.Log("decrease quote asset reserve")
	pool.DecreaseQuoteAssetReserve(sdk.NewInt(100))
	require.Equal(t, sdk.NewInt(999_900), pool.QuoteAssetReserve)

	t.Log("increase quote asset reserve")
	pool.IncreaseQuoteAssetReserve(sdk.NewInt(100))
	require.Equal(t, sdk.NewInt(1_000_000), pool.QuoteAssetReserve)

	t.Log("decrease base asset reserve")
	pool.DecreaseBaseAssetReserve(sdk.NewInt(100))
	require.Equal(t, sdk.NewInt(999_900), pool.BaseAssetReserve)

	t.Log("incrase base asset reserve")
	pool.IncreaseBaseAssetReserve(sdk.NewInt(100))
	require.Equal(t, sdk.NewInt(1_000_000), pool.BaseAssetReserve)
}
