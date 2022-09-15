package keeper

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestCreatePool(t *testing.T) {
	vpoolKeeper, _, ctx := getKeeper(t)

	vpoolKeeper.CreatePool(
		ctx,
		common.Pair_BTC_NUSD,
		sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
		sdk.NewDec(10_000_000),       // 10 tokens
		sdk.NewDec(5_000_000),        // 5 tokens
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
		sdk.MustNewDecFromStr("0.0625"),
		sdk.MustNewDecFromStr("15"),
	)

	exists := vpoolKeeper.ExistsPool(ctx, common.Pair_BTC_NUSD)
	require.True(t, exists)

	notExist := vpoolKeeper.ExistsPool(ctx, common.AssetPair{
		Token0: "BTC",
		Token1: "OTHER",
	})
	require.False(t, notExist)
}

func TestKeeper_GetAllPools(t *testing.T) {
	vpoolKeeper, _, ctx := getKeeper(t)

	var vpools = []*types.VPool{
		{
			Pair:                   common.Pair_BTC_NUSD,
			BaseAssetReserve:       sdk.NewDec(1_000_000),      // 1
			QuoteAssetReserve:      sdk.NewDec(30_000_000_000), // 30,000
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.88"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.20"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.20"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
		{
			Pair:                   common.Pair_ETH_NUSD,
			BaseAssetReserve:       sdk.NewDec(2_000_000),      // 1
			QuoteAssetReserve:      sdk.NewDec(60_000_000_000), // 30,000
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.77"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	}

	for _, vpool := range vpools {
		vpoolKeeper.savePool(ctx, vpool)
	}

	pools := vpoolKeeper.GetAllPools(ctx)
	require.Len(t, pools, 2)
	for _, pool := range pools {
		require.Contains(t, vpools, pool)
	}
}

func TestGetPoolPrices_SetupErrors(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "invalid pair ID on pool",
			test: func(t *testing.T) {
				vpoolWithInvalidPair := types.VPool{
					Pair: common.AssetPair{Token0: "o:o", Token1: "unibi"}}
				vpoolKeeper, _, ctx := getKeeper(t)
				_, err := vpoolKeeper.GetPoolPrices(ctx, vpoolWithInvalidPair)
				require.ErrorContains(t, err, common.ErrInvalidTokenPair.Error())
			},
		},
		{
			name: "attempt to use vpool that hasn't been added",
			test: func(t *testing.T) {
				vpool := types.VPool{Pair: common.MustNewAssetPair("uatom:unibi")}
				vpoolKeeper, _, ctx := getKeeper(t)
				_, err := vpoolKeeper.GetPoolPrices(ctx, vpool)
				require.ErrorContains(t, err, types.ErrPairNotSupported.Error())
			},
		},
		{
			name: "vpool with reserves that don't make sense",
			test: func(t *testing.T) {
				vpool := types.VPool{
					Pair:              common.MustNewAssetPair("uatom:unibi"),
					BaseAssetReserve:  sdk.NewDec(999),
					QuoteAssetReserve: sdk.NewDec(-400),
				}
				vpoolKeeper, _, ctx := getKeeper(t)
				vpoolKeeper.savePool(ctx, &vpool)
				_, err := vpoolKeeper.GetPoolPrices(ctx, vpool)
				require.ErrorContains(t, err, types.ErrNonPositiveReserves.Error())
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, tc.test)
	}
}

func TestGetPoolPrices(t *testing.T) {
	testCases := []struct {
		name               string      // test case name
		vpool              types.VPool // vpool passed to GetPoolPrices
		shouldCreateVpool  bool        // whether to write 'vpool' into the kv store
		mockIndexPrice     sdk.Dec     // indexPriceVal returned by the x/pricefeed keepr
		pricefeedKeeperErr error
		err                error            // An error raised from calling Keeper.GetPoolPrices
		expectedPoolPrices types.PoolPrices // expected output from callign GetPoolPrices
	}{
		{
			name: "happy path - vpool + pricefeed active",
			vpool: types.VPool{
				Pair:                   common.Pair_ETH_NUSD,
				QuoteAssetReserve:      sdk.NewDec(3_000_000), // 3e6
				BaseAssetReserve:       sdk.NewDec(1_000),     // 1e3
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
			shouldCreateVpool: true,
			mockIndexPrice:    sdk.NewDec(99),
			expectedPoolPrices: types.PoolPrices{
				Pair:          common.Pair_ETH_NUSD.String(),
				MarkPrice:     sdk.NewDec(3_000),
				TwapMark:      sdk.NewDec(3_000).String(),
				IndexPrice:    sdk.NewDec(99).String(),
				SwapInvariant: sdk.NewInt(3_000_000_000), // 1e3 * 3e6 = 3e9
				BlockNumber:   2,
			},
		},
		{
			name: "happy path - vpool active, but no index price",
			vpool: types.VPool{
				Pair:                   common.Pair_ETH_NUSD,
				QuoteAssetReserve:      sdk.NewDec(3_000_000), // 3e6
				BaseAssetReserve:       sdk.NewDec(1_000),     // 1e3
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
			shouldCreateVpool:  true,
			mockIndexPrice:     sdk.OneDec().Neg(),
			pricefeedKeeperErr: fmt.Errorf("No index price"),
			expectedPoolPrices: types.PoolPrices{
				Pair:          common.Pair_ETH_NUSD.String(),
				MarkPrice:     sdk.NewDec(3_000),
				TwapMark:      sdk.NewDec(3_000).String(),
				IndexPrice:    sdk.OneDec().Neg().String(),
				SwapInvariant: sdk.NewInt(3_000_000_000), // 1e3 * 3e6 = 3e9
				BlockNumber:   2,
			},
		},
		{
			name: "vpool doesn't exist",
			vpool: types.VPool{
				Pair:                   common.Pair_ETH_NUSD,
				QuoteAssetReserve:      sdk.NewDec(3_000_000), // 3e6
				BaseAssetReserve:       sdk.NewDec(1_000),     // 1e3
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
			shouldCreateVpool: false,
			err:               types.ErrPairNotSupported,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, mocks, ctx := getKeeper(t)
			ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

			if tc.shouldCreateVpool {
				vpoolKeeper.CreatePool(
					ctx,
					tc.vpool.Pair,
					tc.vpool.TradeLimitRatio,
					tc.vpool.QuoteAssetReserve,
					tc.vpool.BaseAssetReserve,
					tc.vpool.FluctuationLimitRatio,
					tc.vpool.MaxOracleSpreadRatio,
					tc.vpool.MaintenanceMarginRatio,
					tc.vpool.MaxLeverage,
				)
			}

			ctx = ctx.WithBlockHeight(2).WithBlockTime(time.Now().Add(5 * time.Second))

			t.Log("mock pricefeedKeeper index price")
			mocks.mockPricefeedKeeper.EXPECT().
				GetCurrentPrice(ctx, tc.vpool.Pair.BaseDenom(), tc.vpool.Pair.QuoteDenom()).
				Return(pftypes.CurrentPrice{
					PairID: tc.vpool.Pair.String(),
					Price:  tc.mockIndexPrice,
				}, tc.pricefeedKeeperErr).
				AnyTimes()

			// logged errors would be called in GetPoolPrices
			var poolPrices types.PoolPrices
			poolPrices, err := vpoolKeeper.GetPoolPrices(ctx, tc.vpool)
			if tc.err != nil {
				assert.ErrorContains(t, err, tc.err.Error())
			} else {
				assert.EqualValues(t, tc.expectedPoolPrices, poolPrices)
			}
		})
	}
}
