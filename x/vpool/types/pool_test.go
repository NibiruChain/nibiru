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
		quoteAmount        sdk.Int
		expectedBaseAmount sdk.Int
	}{
		{
			"quote amount == 0",
			sdk.NewInt(0),
			sdk.NewInt(0),
		},
		{
			"quote amount != 0",
			sdk.NewInt(5_000_000),
			sdk.NewInt(1_666_666),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := NewPool(
				"BTC:NUSD",
				sdk.MustNewDecFromStr("0.9"), // 0.9
				sdk.NewInt(10_000_000),       // 10
				sdk.NewInt(5_000_000),        // 5
				sdk.MustNewDecFromStr("0.1"),
			)

			amount, err := pool.GetBaseAmountByQuoteAmount(Direction_ADD_TO_POOL, tc.quoteAmount)
			require.NoError(t, err)
			require.True(t, amount.Equal(tc.expectedBaseAmount), "expected base: %s, got: %s", tc.expectedBaseAmount.String(), amount.String())
		})
	}
}

func TestGetBaseAmountByQuoteAmount_Error(t *testing.T) {
	tests := []struct {
		name          string
		direction     Direction
		quoteAmount   sdk.Int
		expectedError error
	}{
		{
			"quote after is zero",
			Direction_REMOVE_FROM_POOL,
			sdk.NewInt(10_000_000),
			ErrQuoteReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := NewPool(
				"BTC:NUSD",
				sdk.MustNewDecFromStr("0.9"), // 0.9
				sdk.NewInt(10_000_000),       // 10
				sdk.NewInt(5_000_000),        // 5
				sdk.MustNewDecFromStr("0.1"),
			)

			_, err := pool.GetBaseAmountByQuoteAmount(tc.direction, tc.quoteAmount)
			require.Equal(t, tc.expectedError, err)
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
