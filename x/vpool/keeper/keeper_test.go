package keeper

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestSwapInput_Errors(t *testing.T) {
	tests := []struct {
		name        string
		pair        common.TokenPair
		direction   types.Direction
		quoteAmount sdk.Dec
		baseLimit   sdk.Dec
		error       error
	}{
		{
			"pair not supported",
			"BTC:UST",
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(10),
			sdk.NewDec(10),
			types.ErrPairNotSupported,
		},
		{
			"base amount less than base limit in Long",
			NUSDPair,
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(500_000),
			sdk.NewDec(454_500),
			fmt.Errorf("base amount (238095) is less than selected limit (454500)"),
		},
		{
			"base amount more than base limit in Short",
			NUSDPair,
			types.Direction_REMOVE_FROM_POOL,
			sdk.NewDec(1_000_000),
			sdk.NewDec(454_500),
			fmt.Errorf("base amount (555556) is greater than selected limit (454500)"),
		},
		{
			"quote input bigger than reserve ratio",
			NUSDPair,
			types.Direction_REMOVE_FROM_POOL,
			sdk.NewDec(10_000_000),
			sdk.NewDec(10),
			types.ErrOvertradingLimit,
		},
		{
			"over fluctuation limit fails",
			NUSDPair,
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(1_000_000),
			sdk.NewDec(454544),
			fmt.Errorf("error updating reserve: %w", types.ErrOverFluctuationLimit),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPriceKeeper(gomock.NewController(t)),
			)

			vpoolKeeper.CreatePool(
				ctx,
				NUSDPair,
				sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdk.NewDec(10_000_000),       // 10
				sdk.NewDec(5_000_000),        // 5
				sdk.MustNewDecFromStr("0.1"), // 0.1 fluctuation limit ratio
			)

			_, err := vpoolKeeper.SwapInput(
				ctx,
				tc.pair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
			)
			require.Error(t, err)
		})
	}
}

func TestSwapInput_HappyPath(t *testing.T) {
	tests := []struct {
		name                 string
		direction            types.Direction
		quoteAmount          sdk.Dec
		baseLimit            sdk.Dec
		expectedQuoteReserve sdk.Dec
		expectedBaseReserve  sdk.Dec
		resp                 sdk.Dec
	}{
		{
			"quote amount == 0",
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(0),
			sdk.NewDec(10),
			sdk.NewDec(10_000_000),
			sdk.NewDec(5_000_000),
			sdk.ZeroDec(),
		},
		{
			"normal swap add",
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(1_000_000),
			sdk.NewDec(454_500),
			sdk.NewDec(11_000_000),
			sdk.MustNewDecFromStr("4545454.545454545454545455"),
			sdk.MustNewDecFromStr("454545.454545454545454545"),
		},
		{
			"normal swap remove",
			types.Direction_REMOVE_FROM_POOL,
			sdk.NewDec(1_000_000),
			sdk.NewDec(555_560),
			sdk.NewDec(9_000_000),
			sdk.MustNewDecFromStr("5555555.555555555555555556"),
			sdk.MustNewDecFromStr("555555.555555555555555556"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPriceKeeper(gomock.NewController(t)),
			)

			vpoolKeeper.CreatePool(
				ctx,
				NUSDPair,
				sdk.MustNewDecFromStr("0.9"),  // 0.9 ratio
				sdk.NewDec(10_000_000),        // 10 tokens
				sdk.NewDec(5_000_000),         // 5 tokens
				sdk.MustNewDecFromStr("0.25"), // 0.25 ratio
			)

			res, err := vpoolKeeper.SwapInput(
				ctx,
				NUSDPair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
			)
			require.NoError(t, err)
			require.Equal(t, tc.resp, res)

			pool, err := vpoolKeeper.getPool(ctx, NUSDPair)
			require.NoError(t, err)
			require.Equal(t, tc.expectedQuoteReserve, pool.QuoteAssetReserve)
			require.Equal(t, tc.expectedBaseReserve, pool.BaseAssetReserve)
		})
	}
}
