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

		sdk.NewDec(10_000_000), // 10 tokens
		sdk.NewDec(5_000_000),  // 5 tokens
		types.VpoolConfig{
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
		},
	)

	exists := vpoolKeeper.ExistsPool(ctx, common.Pair_BTC_NUSD)
	require.True(t, exists)

	notExist := vpoolKeeper.ExistsPool(ctx, common.AssetPair{
		Token0: "BTC",
		Token1: "OTHER",
	})
	require.False(t, notExist)
}

func TestEditPoolConfig(t *testing.T) {
	pair := common.Pair_BTC_NUSD
	vpoolStart := types.Vpool{
		Pair:              pair,
		QuoteAssetReserve: sdk.NewDec(10_000_000),
		BaseAssetReserve:  sdk.NewDec(5_000_000),
		Config: types.VpoolConfig{
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
		},
	}

	setupTest := func() (Keeper, sdk.Context) {
		vpoolKeeper, _, ctx := getKeeper(t)
		vpoolKeeper.CreatePool(
			ctx,
			common.Pair_BTC_NUSD,
			vpoolStart.QuoteAssetReserve,
			vpoolStart.BaseAssetReserve,
			vpoolStart.Config,
		)
		exists := vpoolKeeper.ExistsPool(ctx, common.Pair_BTC_NUSD)
		require.True(t, exists)
		return vpoolKeeper, ctx
	}

	testCases := []struct {
		name        string
		newConfig   types.VpoolConfig
		shouldErr   bool
		shouldPanic bool
	}{
		{
			name:      "happy no change to config",
			newConfig: vpoolStart.Config,
			shouldErr: false,
		},
		{
			name:      "happy valid with expected config change",
			newConfig: vpoolStart.Config,
			shouldErr: false,
		},
		{
			name:        "err invalid config nil",
			newConfig:   types.VpoolConfig{},
			shouldPanic: true,
		},
		{
			name: "err invalid config max leverage too high",
			newConfig: types.VpoolConfig{
				// max leverage set too high on purpose
				MaxLeverage:            sdk.MustNewDecFromStr("9001"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
			},
			shouldErr: true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := setupTest()
			if tc.shouldErr {
				err := vpoolKeeper.EditPoolConfig(ctx, pair, tc.newConfig)
				// We expect the initial config if the change fails
				assert.Error(t, err)
				vpool, err := vpoolKeeper.Pools.Get(ctx, pair)
				assert.NoError(t, err)
				assert.EqualValues(t, vpoolStart.Config, vpool.Config)
			} else if tc.shouldPanic {
				require.Panics(t, func() {
					err := vpoolKeeper.EditPoolConfig(ctx, pair, tc.newConfig)
					require.Error(t, err)
				})
			} else {
				err := vpoolKeeper.EditPoolConfig(ctx, pair, tc.newConfig)
				// We expect the new config if the change succeeds
				require.NoError(t, err)
				vpool, err := vpoolKeeper.Pools.Get(ctx, pair)
				assert.NoError(t, err)
				assert.EqualValues(t, tc.newConfig, vpool.Config)
			}
		})
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
				vpoolWithInvalidPair := types.Vpool{
					Pair: common.AssetPair{Token0: "o:o", Token1: "unibi"}}
				vpoolKeeper, _, ctx := getKeeper(t)
				_, err := vpoolKeeper.GetPoolPrices(ctx, vpoolWithInvalidPair)
				require.ErrorContains(t, err, common.ErrInvalidTokenPair.Error())
			},
		},
		{
			name: "attempt to use vpool that hasn't been added",
			test: func(t *testing.T) {
				vpool := types.Vpool{Pair: common.MustNewAssetPair("uatom:unibi")}
				vpoolKeeper, _, ctx := getKeeper(t)
				_, err := vpoolKeeper.GetPoolPrices(ctx, vpool)
				require.ErrorContains(t, err, types.ErrPairNotSupported.Error())
			},
		},
		{
			name: "vpool with reserves that don't make sense",
			test: func(t *testing.T) {
				vpool := types.Vpool{
					Pair:              common.MustNewAssetPair("uatom:unibi"),
					BaseAssetReserve:  sdk.NewDec(999),
					QuoteAssetReserve: sdk.NewDec(-400),
				}
				vpoolKeeper, _, ctx := getKeeper(t)
				vpoolKeeper.Pools.Insert(ctx, vpool.Pair, vpool)
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
		vpool              types.Vpool // vpool passed to GetPoolPrices
		shouldCreateVpool  bool        // whether to write 'vpool' into the kv store
		mockIndexPrice     sdk.Dec     // indexPriceVal returned by the x/pricefeed keepr
		pricefeedKeeperErr error
		err                error            // An error raised from calling Keeper.GetPoolPrices
		expectedPoolPrices types.PoolPrices // expected output from callign GetPoolPrices
	}{
		{
			name: "happy path - vpool + pricefeed active",
			vpool: types.Vpool{
				Pair:              common.Pair_ETH_NUSD,
				QuoteAssetReserve: sdk.NewDec(3_000_000), // 3e6
				BaseAssetReserve:  sdk.NewDec(1_000),     // 1e3
				Config: types.VpoolConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
					TradeLimitRatio:        sdk.OneDec(),
				},
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
			vpool: types.Vpool{
				Pair:              common.Pair_ETH_NUSD,
				QuoteAssetReserve: sdk.NewDec(3_000_000), // 3e6
				BaseAssetReserve:  sdk.NewDec(1_000),     // 1e3
				Config: types.VpoolConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
					TradeLimitRatio:        sdk.OneDec(),
				},
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
			vpool: types.Vpool{
				Pair:              common.Pair_ETH_NUSD,
				QuoteAssetReserve: sdk.NewDec(3_000_000), // 3e6
				BaseAssetReserve:  sdk.NewDec(1_000),     // 1e3
				Config: types.VpoolConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
				},
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
					tc.vpool.QuoteAssetReserve,
					tc.vpool.BaseAssetReserve,
					tc.vpool.Config,
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
