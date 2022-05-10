package keeper

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSwapInput_Errors(t *testing.T) {
	tests := []struct {
		name        string
		pair        common.TokenPair
		direction   types.Direction
		quoteAmount sdk.Int
		baseLimit   sdk.Int
		error       error
	}{
		{
			"pair not supported",
			"BTC:UST",
			types.Direction_ADD_TO_POOL,
			sdk.NewInt(10),
			sdk.NewInt(10),
			types.ErrPairNotSupported,
		},
		{
			"base amount less than base limit in Long",
			NUSDPair,
			types.Direction_ADD_TO_POOL,
			sdk.NewInt(500_000),
			sdk.NewInt(454_500),
			fmt.Errorf("base amount (238095) is less than selected limit (454500)"),
		},
		{
			"base amount more than base limit in Short",
			NUSDPair,
			types.Direction_REMOVE_FROM_POOL,
			sdk.NewInt(1_000_000),
			sdk.NewInt(454_500),
			fmt.Errorf("base amount (555556) is greater than selected limit (454500)"),
		},
		{
			"quote input bigger than reserve ratio",
			NUSDPair,
			types.Direction_REMOVE_FROM_POOL,
			sdk.NewInt(10_000_000),
			sdk.NewInt(10),
			types.ErrOvertradingLimit,
		},
		{
			"over fluctuation limit fails",
			NUSDPair,
			types.Direction_ADD_TO_POOL,
			sdk.NewInt(1_000_000),
			sdk.NewInt(454544),
			fmt.Errorf("error updating reserve: %w", types.ErrOverFluctuationLimit),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper, ctx := AmmKeeper(t)

			err := keeper.CreatePool(
				ctx,
				NUSDPair,
				sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdk.NewInt(10_000_000),       // 10
				sdk.NewInt(5_000_000),        // 5
				sdk.MustNewDecFromStr("0.1"), // 0.1 fluctuation limit ratio
			)
			require.NoError(t, err)

			_, err = keeper.SwapInput(
				ctx,
				tc.pair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
			)
			require.EqualError(t, err, tc.error.Error())
		})
	}
}

func TestSwapInput_HappyPath(t *testing.T) {
	tests := []struct {
		name                 string
		direction            types.Direction
		quoteAmount          sdk.Int
		baseLimit            sdk.Int
		expectedQuoteReserve sdk.Int
		expectedBaseReserve  sdk.Int
		resp                 sdk.Int
	}{
		{
			"quote amount == 0",
			types.Direction_ADD_TO_POOL,
			sdk.NewInt(0),
			sdk.NewInt(10),
			sdk.NewInt(10_000_000),
			sdk.NewInt(5_000_000),
			sdk.ZeroInt(),
		},
		{
			"normal swap add",
			types.Direction_ADD_TO_POOL,
			sdk.NewInt(1_000_000),
			sdk.NewInt(454_500),
			sdk.NewInt(11_000_000),
			sdk.NewInt(4_545_455),
			sdk.NewInt(454_545),
		},
		{
			"normal swap remove",
			types.Direction_REMOVE_FROM_POOL,
			sdk.NewInt(1_000_000),
			sdk.NewInt(555_560),
			sdk.NewInt(9_000_000),
			sdk.NewInt(5_555_556),
			sdk.NewInt(555_556),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper, ctx := AmmKeeper(t)

			err := keeper.CreatePool(
				ctx,
				NUSDPair,
				sdk.MustNewDecFromStr("0.9"),  // 0.9 ratio
				sdk.NewInt(10_000_000),        // 10 tokens
				sdk.NewInt(5_000_000),         // 5 tokens
				sdk.MustNewDecFromStr("0.25"), // 0.25 ratio
			)
			require.NoError(t, err)

			res, err := keeper.SwapInput(
				ctx,
				NUSDPair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
			)
			require.NoError(t, err)
			require.Equal(t, res, tc.resp)

			pool, err := keeper.getPool(ctx, NUSDPair)
			require.NoError(t, err)
			require.Equal(t, tc.expectedQuoteReserve, pool.QuoteAssetReserve)
			require.Equal(t, tc.expectedBaseReserve, pool.BaseAssetReserve)
		})
	}
}

func TestCreatePool(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)

	err := ammKeeper.CreatePool(
		ctx,
		NUSDPair,
		sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
		sdk.NewInt(10_000_000),       // 10 tokens
		sdk.NewInt(5_000_000),        // 5 tokens
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
	)
	require.NoError(t, err)

	exists := ammKeeper.existsPool(ctx, NUSDPair)
	require.True(t, exists)

	notExist := ammKeeper.existsPool(ctx, "BTC:OTHER")
	require.False(t, notExist)
}
