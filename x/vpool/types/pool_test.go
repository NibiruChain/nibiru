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
	isEnough, err := pool.HasEnoughQuoteReserve(sdk.NewInt(8_000_000))
	require.NoError(t, err)
	require.True(t, isEnough)

	// equal to ratio limit
	isEnough, err = pool.HasEnoughQuoteReserve(sdk.NewInt(9_000_000))
	require.NoError(t, err)
	require.True(t, isEnough)

	// more than ratio limit
	isEnough, err = pool.HasEnoughQuoteReserve(sdk.NewInt(9_000_001))
	require.NoError(t, err)
	require.False(t, isEnough)
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
			sdk.NewInt(1_666_665),
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

			amount, err := pool.GetBaseAmountByQuoteAmount(Direction_ADD_TO_AMM, tc.quoteAmount)
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
			Direction_REMOVE_FROM_AMM,
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

func TestIncreaseToken0Reserve(t *testing.T) {
	pool := NewPool(
		"ATOM:NUSD",
		sdk.MustNewDecFromStr("0.9"),
		sdk.NewInt(1_000_000),
		sdk.NewInt(1_000_000),
		sdk.MustNewDecFromStr("0.1"),
	)

	pool.IncreaseToken0Reserve(sdk.NewInt(100))

	quoteAssetReserve, err := pool.GetPoolToken0ReserveAsInt()
	require.NoError(t, err)

	require.Equal(t, sdk.NewInt(1_000_100), quoteAssetReserve)
}

func TestIncreaseDecreaseReserves(t *testing.T) {
	pool := NewPool(
		"ATOM:NUSD",
		sdk.MustNewDecFromStr("0.9"),
		sdk.NewInt(1_000_000),
		sdk.NewInt(1_000_000),
		sdk.MustNewDecFromStr("0.1"),
	)

	// DecreaseToken1
	pool.DecreaseToken1Reserve(sdk.NewInt(100))
	baseAssetReserve, err := pool.GetPoolToken1ReserveAsInt()
	require.NoError(t, err)
	require.Equal(t, sdk.NewInt(999_900), baseAssetReserve)

	// Increase Base Asset
	pool.IncreaseToken1Reserve(sdk.NewInt(100))
	baseAssetReserve, err = pool.GetPoolToken1ReserveAsInt()
	require.NoError(t, err)
	require.Equal(t, sdk.NewInt(1_000_000), baseAssetReserve)

	// DecreaseToken0
	pool.DecreaseToken0Reserve(sdk.NewInt(100))
	quoteAssetReserve, err := pool.GetPoolToken0ReserveAsInt()
	require.NoError(t, err)
	require.Equal(t, sdk.NewInt(999_900), quoteAssetReserve)

	// Increase Quote Asset
	pool.IncreaseToken0Reserve(sdk.NewInt(100))
	quoteAssetReserve, err = pool.GetPoolToken0ReserveAsInt()
	require.NoError(t, err)
	require.Equal(t, sdk.NewInt(1_000_000), quoteAssetReserve)
}
