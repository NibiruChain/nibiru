package keeper

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestCreatePool(t *testing.T) {
	vpoolKeeper, _, ctx := getKeeper(t)

	assert.NoError(t, vpoolKeeper.CreatePool(
		ctx,
		/* pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		/* quote */ sdk.NewDec(10*common.TO_MICRO), // 10 tokens
		/* base */ sdk.NewDec(5*common.TO_MICRO), // 5 tokens
		types.VpoolConfig{
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
		},
		sdk.ZeroDec(),
		sdk.OneDec(),
	))

	exists := vpoolKeeper.ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
	require.True(t, exists)

	notExist := vpoolKeeper.ExistsPool(ctx, "BTC:OTHER")
	require.False(t, notExist)
}

func TestEditPoolConfig(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	vpoolStart := types.Vpool{
		Pair:              pair,
		QuoteAssetReserve: sdk.NewDec(10 * common.TO_MICRO),
		BaseAssetReserve:  sdk.NewDec(5 * common.TO_MICRO),
		SqrtDepth:         common.MustSqrtDec(sdk.NewDec(5 * 10 * common.TO_MICRO * common.TO_MICRO)),
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
		assert.NoError(t, vpoolKeeper.CreatePool(
			ctx,
			asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			vpoolStart.QuoteAssetReserve,
			vpoolStart.BaseAssetReserve,
			vpoolStart.Config,
			sdk.ZeroDec(),
			sdk.OneDec(),
		))
		exists := vpoolKeeper.ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
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
				vpoolWithInvalidPair := types.Vpool{Pair: "o:o:unibi"}
				vpoolKeeper, _, ctx := getKeeper(t)
				_, err := vpoolKeeper.GetPoolPrices(ctx, vpoolWithInvalidPair)
				require.ErrorContains(t, err, asset.ErrInvalidTokenPair.Error())
			},
		},
		{
			name: "attempt to use vpool that hasn't been added",
			test: func(t *testing.T) {
				vpool := types.Vpool{Pair: asset.MustNewPair("uatom:unibi")}
				vpoolKeeper, _, ctx := getKeeper(t)
				_, err := vpoolKeeper.GetPoolPrices(ctx, vpool)
				require.ErrorContains(t, err, types.ErrPairNotSupported.Error())
			},
		},
		{
			name: "vpool with reserves that don't make sense",
			test: func(t *testing.T) {
				vpool := types.Vpool{
					Pair:              asset.MustNewPair("uatom:unibi"),
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
		oracleKeeperErr    error
		err                error            // An error raised from calling Keeper.GetPoolPrices
		expectedPoolPrices types.PoolPrices // expected output from callign GetPoolPrices
	}{
		{
			name: "happy path - vpool + pricefeed active",
			vpool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				QuoteAssetReserve: sdk.NewDec(3 * common.TO_MICRO), // 3e6
				BaseAssetReserve:  sdk.NewDec(1_000),               // 1e3
				SqrtDepth:         common.MustSqrtDec(sdk.NewDec(3_000 * common.TO_MICRO)),
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
				Pair:          asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				MarkPrice:     sdk.NewDec(3_000),
				TwapMark:      sdk.NewDec(3_000).String(),
				IndexPrice:    sdk.NewDec(99).String(),
				SwapInvariant: sdk.NewInt(3_000 * common.TO_MICRO), // 1e3 * 3e6 = 3e9
				BlockNumber:   2,
			},
		},
		{
			name: "happy path - vpool active, but no index price",
			vpool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				QuoteAssetReserve: sdk.NewDec(3 * common.TO_MICRO), // 3e6
				BaseAssetReserve:  sdk.NewDec(1_000),               // 1e3
				SqrtDepth:         common.MustSqrtDec(sdk.NewDec(3_000 * common.TO_MICRO)),
				Config: types.VpoolConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
					TradeLimitRatio:        sdk.OneDec(),
				},
			},
			shouldCreateVpool: true,
			mockIndexPrice:    sdk.OneDec().Neg(),
			oracleKeeperErr:   fmt.Errorf("No index price"),
			expectedPoolPrices: types.PoolPrices{
				Pair:          asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				MarkPrice:     sdk.NewDec(3_000),
				TwapMark:      sdk.NewDec(3_000).String(),
				IndexPrice:    sdk.OneDec().Neg().String(),
				SwapInvariant: sdk.NewInt(3_000 * common.TO_MICRO), // 1e3 * 3e6 = 3e9
				BlockNumber:   2,
			},
		},
		{
			name: "vpool doesn't exist",
			vpool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				QuoteAssetReserve: sdk.NewDec(3 * common.TO_MICRO), // 3e6
				BaseAssetReserve:  sdk.NewDec(1_000),               // 1e3
				SqrtDepth:         common.MustSqrtDec(sdk.NewDec(3_000 * common.TO_MICRO)),
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
				assert.NoError(t, vpoolKeeper.CreatePool(
					ctx,
					tc.vpool.Pair,
					tc.vpool.QuoteAssetReserve,
					tc.vpool.BaseAssetReserve,
					tc.vpool.Config,
					sdk.ZeroDec(),
					sdk.OneDec(),
				))
			}

			ctx = ctx.WithBlockHeight(2).WithBlockTime(time.Now().Add(5 * time.Second))

			t.Log("mock oracleKeeper index price")
			mocks.mockOracleKeeper.EXPECT().
				GetExchangeRate(ctx, tc.vpool.Pair).
				Return(tc.mockIndexPrice, tc.oracleKeeperErr).
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

func TestEditSwapInvariant(t *testing.T) {
	pair := asset.Registry.Pair(denoms.NIBI, denoms.NUSD)
	vpoolStart := types.Vpool{
		Pair:              pair,
		QuoteAssetReserve: sdk.NewDec(10 * common.TO_MICRO),
		BaseAssetReserve:  sdk.NewDec(5 * common.TO_MICRO),
		SqrtDepth:         common.MustSqrtDec(sdk.NewDec(5 * 10 * common.TO_MICRO * common.TO_MICRO)),
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
		assert.NoError(t, vpoolKeeper.CreatePool(
			ctx,
			pair,
			vpoolStart.QuoteAssetReserve,
			vpoolStart.BaseAssetReserve,
			vpoolStart.Config,
			sdk.ZeroDec(),
			sdk.OneDec(),
		))
		exists := vpoolKeeper.ExistsPool(ctx, pair)
		require.True(t, exists)
		return vpoolKeeper, ctx
	}

	type Reserves struct {
		Base  sdk.Dec
		Quote sdk.Dec
	}

	testCases := []struct {
		name                    string
		swapInvariantMultiplier sdk.Dec
		newReserves             Reserves
		shouldErr               bool
		shouldPanic             bool
	}{
		{
			name:                    "happy reserves increase 2x",
			swapInvariantMultiplier: sdk.NewDec(4),
			newReserves: Reserves{
				Base:  vpoolStart.BaseAssetReserve.MulInt64(2),
				Quote: vpoolStart.QuoteAssetReserve.MulInt64(2)},
			shouldErr: false,
		},
		{
			name:                    "happy no change",
			swapInvariantMultiplier: sdk.NewDec(1),
			newReserves: Reserves{
				Base:  vpoolStart.BaseAssetReserve,
				Quote: vpoolStart.QuoteAssetReserve},
			shouldErr: false,
		},
		{
			name:                    "happy reserves increase 500x",
			swapInvariantMultiplier: sdk.NewDec(250_000), // 500**2
			newReserves: Reserves{
				Base:  vpoolStart.BaseAssetReserve.MulInt64(500),
				Quote: vpoolStart.QuoteAssetReserve.MulInt64(500)},
			shouldErr: false,
		},
		{
			name:                    "happy reserves shrink 2x",
			swapInvariantMultiplier: sdk.MustNewDecFromStr("0.25"), // (1/2)**2
			newReserves: Reserves{
				Base:  vpoolStart.BaseAssetReserve.QuoInt64(2),
				Quote: vpoolStart.QuoteAssetReserve.QuoInt64(2)},
			shouldErr: false,
		},
		{
			name:                    "happy reserves shrink 100x",
			swapInvariantMultiplier: sdk.MustNewDecFromStr("0.0001"), // (1/100)**2
			newReserves: Reserves{
				Base:  vpoolStart.BaseAssetReserve.QuoInt64(100),
				Quote: vpoolStart.QuoteAssetReserve.QuoInt64(100)},
			shouldErr: false,
		},
		{
			name:                    "err invalid multiplier",
			swapInvariantMultiplier: sdk.Dec{},
			shouldErr:               true,
		},
		{
			name:                    "err invariant zero causes zero reserves",
			swapInvariantMultiplier: sdk.NewDec(0),
			shouldErr:               true,
		},
		{
			name:                    "err invariant negative",
			swapInvariantMultiplier: sdk.NewDec(-10),
			shouldErr:               true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := setupTest()
			if tc.shouldErr {
				err := vpoolKeeper.EditSwapInvariant(ctx,
					types.EditSwapInvariantsProposal_SwapInvariantMultiple{
						Pair: pair, Multiplier: tc.swapInvariantMultiplier,
					})
				// We expect the initial config if the change fails
				assert.Error(t, err)
				vpool, err := vpoolKeeper.Pools.Get(ctx, pair)
				assert.NoError(t, err)
				assert.EqualValues(t, vpoolStart.BaseAssetReserve, vpool.BaseAssetReserve)
				assert.EqualValues(t, vpoolStart.QuoteAssetReserve, vpool.QuoteAssetReserve)
			} else if tc.shouldPanic {
				require.Panics(t, func() {
					err := vpoolKeeper.EditSwapInvariant(ctx,
						types.EditSwapInvariantsProposal_SwapInvariantMultiple{
							Pair: pair, Multiplier: tc.swapInvariantMultiplier,
						})
					require.Error(t, err)
				})
			} else {
				err := vpoolKeeper.EditSwapInvariant(ctx,
					types.EditSwapInvariantsProposal_SwapInvariantMultiple{
						Pair: pair, Multiplier: tc.swapInvariantMultiplier,
					})
				// We expect the new config if the change succeeds
				require.NoError(t, err)
				vpool, err := vpoolKeeper.Pools.Get(ctx, pair)
				assert.NoError(t, err)
				assert.EqualValues(t, tc.newReserves.Base, vpool.BaseAssetReserve)
				assert.EqualValues(t, tc.newReserves.Quote, vpool.QuoteAssetReserve)
			}
		})
	}
}
