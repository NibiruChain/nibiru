package keeper

import (
	"testing"

	ammtypes "github.com/MatrixDao/matrix/x/vamm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSwapInput_Errors(t *testing.T) {
	tests := []struct {
		name        string
		pair        string
		direction   ammtypes.Direction
		quoteAmount sdktypes.Int
		baseLimit   sdktypes.Int
		error       error
	}{
		{
			"pair not supported",
			"BTC:UST",
			ammtypes.Direction_ADD_TO_AMM,
			sdktypes.NewInt(10),
			sdktypes.NewInt(10),
			ammtypes.ErrPairNotSupported,
		},
		{
			"quote input bigger than reserve ratio",
			UsdmPair,
			ammtypes.Direction_REMOVE_FROM_AMM,
			sdktypes.NewInt(10_000_000),
			sdktypes.NewInt(10),
			ammtypes.ErrOvertradingLimit,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper, ctx := AmmKeeper(t)

			err := keeper.CreatePool(
				ctx,
				UsdmPair,
				sdktypes.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdktypes.NewInt(10_000_000),       // 10
				sdktypes.NewInt(5_000_000),        // 5
			)
			require.NoError(t, err)

			_, err = keeper.SwapInput(
				ctx,
				tc.pair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
				false,
			)
			require.EqualError(t, err, tc.error.Error())
		})
	}
}

func TestSwapInput_HappyPath(t *testing.T) {
	tests := []struct {
		name                 string
		direction            ammtypes.Direction
		quoteAmount          sdktypes.Int
		baseLimit            sdktypes.Int
		expectedQuoteReserve sdktypes.Int
		expectedBaseReserve  sdktypes.Int
		resp                 sdktypes.Int
	}{
		{
			"quote amount == 0",
			ammtypes.Direction_ADD_TO_AMM,
			sdktypes.NewInt(0),
			sdktypes.NewInt(10),
			sdktypes.NewInt(10_000_000),
			sdktypes.NewInt(5_000_000),
			sdktypes.ZeroInt(),
		},
		{
			"normal swap add",
			ammtypes.Direction_ADD_TO_AMM,
			sdktypes.NewInt(1_000_000),
			sdktypes.NewInt(454_500),
			sdktypes.NewInt(11_000_000),
			sdktypes.NewInt(4_545_456),
			sdktypes.NewInt(454_544),
		},
		{
			"normal swap remove",
			ammtypes.Direction_REMOVE_FROM_AMM,
			sdktypes.NewInt(1_000_000),
			sdktypes.NewInt(555_560),
			sdktypes.NewInt(9_000_000),
			sdktypes.NewInt(5_555_556),
			sdktypes.NewInt(555_556),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper, ctx := AmmKeeper(t)

			err := keeper.CreatePool(
				ctx,
				UsdmPair,
				sdktypes.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdktypes.NewInt(10_000_000),       // 10 tokens
				sdktypes.NewInt(5_000_000),        // 5 tokens
			)
			require.NoError(t, err)

			res, err := keeper.SwapInput(
				ctx,
				UsdmPair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
				false,
			)
			require.NoError(t, err)
			require.Equal(t, res, tc.resp)

			pool, err := keeper.getPool(ctx, UsdmPair)
			quoteAmount, err := pool.GetPoolQuoteAssetReserveAsInt()
			require.NoError(t, err)
			require.Equal(t, tc.expectedQuoteReserve, quoteAmount)

			baseAmount, err := pool.GetPoolBaseAssetReserveAsInt()
			require.NoError(t, err)
			require.Equal(t, tc.expectedBaseReserve, baseAmount)
		})
	}
}

func TestCreatePool(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)

	err := ammKeeper.CreatePool(
		ctx,
		UsdmPair,
		sdktypes.MustNewDecFromStr("0.9"), // 0.9 ratio
		sdktypes.NewInt(10_000_000),       // 10 tokens
		sdktypes.NewInt(5_000_000),        // 5 tokens
	)
	require.NoError(t, err)

	exists := ammKeeper.existsPool(ctx, UsdmPair)
	require.True(t, exists)

	notExist := ammKeeper.existsPool(ctx, "BTC:OTHER")
	require.False(t, notExist)
}
