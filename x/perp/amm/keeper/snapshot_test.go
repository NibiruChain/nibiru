package keeper

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

func TestGetSnapshotPrice(t *testing.T) {
	tests := []struct {
		name          string
		pair          asset.Pair
		quoteReserve  sdk.Dec
		baseReserve   sdk.Dec
		PegMultiplier sdk.Dec
		Bias          sdk.Dec

		twapCalcOption types.TwapCalcOption
		direction      types.Direction
		assetAmount    sdk.Dec
		expectedPrice  sdk.Dec
	}{
		{
			name:           "spot price calc",
			pair:           asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:   sdk.NewDec(1_000),
			baseReserve:    sdk.NewDec(1_000),
			PegMultiplier:  sdk.NewDec(2),
			Bias:           sdk.ZeroDec(),
			twapCalcOption: types.TwapCalcOption_SPOT,
			expectedPrice:  sdk.NewDec(2),
		},
		{
			name:           "spot price calc with bias",
			pair:           asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:   sdk.NewDec(1_200),
			baseReserve:    sdk.NewDec(1_000),
			PegMultiplier:  sdk.NewDec(2),
			Bias:           sdk.ZeroDec(),
			twapCalcOption: types.TwapCalcOption_SPOT,
			expectedPrice:  sdk.MustNewDecFromStr("2.4"), // 1200/1000*2
		},
		{
			name:           "quote asset swap add to pool calc",
			pair:           asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:   sdk.NewDec(1_000),
			baseReserve:    sdk.NewDec(1_000),
			PegMultiplier:  sdk.NewDec(3),
			Bias:           sdk.ZeroDec(),
			twapCalcOption: types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:      types.Direction_LONG,
			assetAmount:    sdk.NewDec(1),
			expectedPrice:  sdk.MustNewDecFromStr("2.997002997002997003"), //almost 3
		},
		{
			name:           "quote asset swap add to pool calc with bias",
			pair:           asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:   sdk.NewDec(1_000),
			baseReserve:    sdk.NewDec(1_000),
			PegMultiplier:  sdk.NewDec(3),
			Bias:           sdk.NewDec(1_000),
			twapCalcOption: types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:      types.Direction_LONG,
			assetAmount:    sdk.NewDec(1),
			expectedPrice:  sdk.MustNewDecFromStr("11.976047904191616765"), // 3 * (2,000 - 2,000,000 / 1,002)
		},
		{
			name:           "quote asset swap remove from pool calc",
			pair:           asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:   sdk.NewDec(1_000),
			baseReserve:    sdk.NewDec(1_000),
			Bias:           sdk.NewDec(1_000),
			PegMultiplier:  sdk.NewDec(3),
			twapCalcOption: types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:      types.Direction_SHORT,
			assetAmount:    sdk.NewDec(1),
			expectedPrice:  sdk.MustNewDecFromStr("12.024048096192384771"), // 3 * (2,000 - 2,000,000 / 998)
		},
		{
			name:           "base asset swap add to pool calc",
			pair:           asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:   sdk.NewDec(1_000),
			baseReserve:    sdk.NewDec(1_000),
			twapCalcOption: types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:      types.Direction_LONG,
			Bias:           sdk.NewDec(1_000),
			PegMultiplier:  sdk.NewDec(3),
			assetAmount:    sdk.NewDec(1),
			expectedPrice:  sdk.MustNewDecFromStr("0.083291687489588539"), // (1,000,000 / 2000 - 1,000,000 / 2001) * 1/3
			// 1 / expected price ~= 12.006
		},
		{
			name:           "base asset swap remove to pool calc",
			pair:           asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:   sdk.NewDec(1_000),
			baseReserve:    sdk.NewDec(1_000),
			twapCalcOption: types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:      types.Direction_SHORT,
			Bias:           sdk.NewDec(1_000),
			PegMultiplier:  sdk.NewDec(3),
			assetAmount:    sdk.NewDec(1),
			expectedPrice:  sdk.MustNewDecFromStr("0.083375020843755211"), // (1,000,000 / 2000 - 1,000,000 / 1999) * 1/3
			// 1 / expected price ~= 11.994
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			snapshot := types.NewReserveSnapshot(
				tc.pair,
				tc.baseReserve,
				tc.quoteReserve,
				tc.PegMultiplier,
				tc.Bias,
				time.Now(),
			)

			snapshotPriceOpts := snapshotPriceOptions{
				pair:           tc.pair,
				twapCalcOption: tc.twapCalcOption,
				direction:      tc.direction,
				assetAmount:    tc.assetAmount,
			}

			price, err := getPriceWithSnapshot(
				snapshot,
				snapshotPriceOpts,
			)

			fmt.Println("price", price)

			require.NoError(t, err)
			require.EqualValuesf(t, tc.expectedPrice, price,
				"expected %s, got %s", tc.expectedPrice.String(), price.String())
		})
	}
}
