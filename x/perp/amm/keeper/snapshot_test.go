package keeper

import (
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
		name              string
		pair              asset.Pair
		quoteAssetReserve sdk.Dec
		baseAssetReserve  sdk.Dec
		twapCalcOption    types.TwapCalcOption
		direction         types.Direction
		assetAmount       sdk.Dec
		expectedPrice     sdk.Dec
	}{
		{
			name:              "spot price calc",
			pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteAssetReserve: sdk.NewDec(40_000),
			baseAssetReserve:  sdk.NewDec(2),
			twapCalcOption:    types.TwapCalcOption_SPOT,
			expectedPrice:     sdk.NewDec(20_000),
		},
		{
			name:              "quote asset swap add to pool calc",
			pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:         types.Direction_LONG,
			assetAmount:       sdk.NewDec(3_000),
			expectedPrice:     sdk.NewDec(500),
		},
		{
			name:              "quote asset swap remove from pool calc",
			pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:         types.Direction_SHORT,
			assetAmount:       sdk.NewDec(1_500),
			expectedPrice:     sdk.NewDec(1_000),
		},
		{
			name:              "base asset swap add to pool calc",
			pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:         types.Direction_LONG,
			assetAmount:       sdk.NewDec(500),
			expectedPrice:     sdk.NewDec(1_000),
		},
		{
			name:              "base asset swap remove from pool calc",
			pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:         types.Direction_SHORT,
			assetAmount:       sdk.NewDec(500),
			expectedPrice:     sdk.NewDec(3_000),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			snapshot := types.NewReserveSnapshot(
				tc.pair,
				tc.baseAssetReserve,
				tc.quoteAssetReserve,
				sdk.NewDec(1),
				sdk.ZeroDec(),
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

			require.NoError(t, err)
			require.EqualValuesf(t, tc.expectedPrice, price,
				"expected %s, got %s", tc.expectedPrice.String(), price.String())
		})
	}
}
