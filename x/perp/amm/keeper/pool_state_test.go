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
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

func TestCreatePool(t *testing.T) {
	perpammKeeper, _, ctx := getKeeper(t)

	require.NoError(t, perpammKeeper.CreatePool(
		ctx,
		/* pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		/* quote */ sdk.NewDec(5*common.TO_MICRO), // 10 tokens
		/* base */ sdk.NewDec(5*common.TO_MICRO), // 5 tokens
		types.MarketConfig{
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
		},
		sdk.ZeroDec(),
		sdk.NewDec(2),
	))

	exists := perpammKeeper.ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
	require.True(t, exists)

	notExist := perpammKeeper.ExistsPool(ctx, "BTC:OTHER")
	require.False(t, notExist)
}

func TestEditPoolConfig(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	marketStart := types.Market{
		Pair:          pair,
		QuoteReserve:  sdk.NewDec(5 * common.TO_MICRO),
		BaseReserve:   sdk.NewDec(5 * common.TO_MICRO),
		PegMultiplier: sdk.NewDec(2),
		SqrtDepth:     common.MustSqrtDec(sdk.NewDec(5 * 10 * common.TO_MICRO * common.TO_MICRO)),
		Config: types.MarketConfig{
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
		},
	}

	setupTest := func() (Keeper, sdk.Context) {
		perpammKeeper, _, ctx := getKeeper(t)
		require.NoError(t, perpammKeeper.CreatePool(
			ctx,
			asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			marketStart.QuoteReserve,
			marketStart.BaseReserve,
			marketStart.Config,
			sdk.ZeroDec(),
			sdk.OneDec(),
		))
		exists := perpammKeeper.ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
		require.True(t, exists)
		return perpammKeeper, ctx
	}

	testCases := []struct {
		name        string
		newConfig   types.MarketConfig
		shouldErr   bool
		shouldPanic bool
	}{
		{
			name:      "happy no change to config",
			newConfig: marketStart.Config,
			shouldErr: false,
		},
		{
			name:      "happy valid with expected config change",
			newConfig: marketStart.Config,
			shouldErr: false,
		},
		{
			name:        "err invalid config nil",
			newConfig:   types.MarketConfig{},
			shouldPanic: true,
		},
		{
			name: "err invalid config max leverage too high",
			newConfig: types.MarketConfig{
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
			perpammKeeper, ctx := setupTest()
			if tc.shouldErr {
				err := perpammKeeper.EditPoolConfig(ctx, pair, tc.newConfig)
				// We expect the initial config if the change fails
				assert.Error(t, err)
				market, err := perpammKeeper.Pools.Get(ctx, pair)
				assert.NoError(t, err)
				assert.EqualValues(t, marketStart.Config, market.Config)
			} else if tc.shouldPanic {
				require.Panics(t, func() {
					err := perpammKeeper.EditPoolConfig(ctx, pair, tc.newConfig)
					require.Error(t, err)
				})
			} else {
				err := perpammKeeper.EditPoolConfig(ctx, pair, tc.newConfig)
				// We expect the new config if the change succeeds
				require.NoError(t, err)
				market, err := perpammKeeper.Pools.Get(ctx, pair)
				assert.NoError(t, err)
				assert.EqualValues(t, tc.newConfig, market.Config)
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
				marketWithInvalidPair := types.Market{Pair: "o:o:unibi"}
				perpammKeeper, _, ctx := getKeeper(t)
				_, err := perpammKeeper.GetPoolPrices(ctx, marketWithInvalidPair)
				require.ErrorContains(t, err, asset.ErrInvalidTokenPair.Error())
			},
		},
		{
			name: "attempt to use market that hasn't been added",
			test: func(t *testing.T) {
				market := types.Market{Pair: asset.MustNewPair("uatom:unibi")}
				perpammKeeper, _, ctx := getKeeper(t)
				_, err := perpammKeeper.GetPoolPrices(ctx, market)
				require.ErrorContains(t, err, types.ErrPairNotSupported.Error())
			},
		},
		{
			name: "market with reserves that don't make sense",
			test: func(t *testing.T) {
				market := types.Market{
					Pair:         asset.MustNewPair("uatom:unibi"),
					BaseReserve:  sdk.NewDec(999),
					QuoteReserve: sdk.NewDec(-400),
				}
				perpammKeeper, _, ctx := getKeeper(t)
				perpammKeeper.Pools.Insert(ctx, market.Pair, market)
				_, err := perpammKeeper.GetPoolPrices(ctx, market)
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
		name               string       // test case name
		market             types.Market // market passed to GetPoolPrices
		shouldCreateMarket bool         // whether to write 'market' into the kv store
		mockIndexPrice     sdk.Dec      // indexPriceVal returned by the x/pricefeed keepr
		oracleKeeperErr    error
		err                error            // An error raised from calling Keeper.GetPoolPrices
		expectedPoolPrices types.PoolPrices // expected output from callign GetPoolPrices
	}{
		{
			name: "happy path - market + pricefeed active",
			market: types.Market{
				Pair:         asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				QuoteReserve: sdk.NewDec(1_000), // 3e6
				BaseReserve:  sdk.NewDec(1_000), // 1e3
				SqrtDepth:    common.MustSqrtDec(sdk.NewDec(1_000 * 1_000)),
				Config: types.MarketConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
					TradeLimitRatio:        sdk.OneDec(),
				},
				PegMultiplier: sdk.NewDec(3_000),
			},
			shouldCreateMarket: true,
			mockIndexPrice:     sdk.NewDec(99),
			expectedPoolPrices: types.PoolPrices{
				Pair:          asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				MarkPrice:     sdk.NewDec(3_000),
				TwapMark:      sdk.NewDec(3_000).String(),
				IndexPrice:    sdk.NewDec(99).String(),
				SwapInvariant: sdk.NewInt(1_000 * 1_000),
				BlockNumber:   2,
			},
		},
		{
			name: "happy path - market active, but no index price",
			market: types.Market{
				Pair:         asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				QuoteReserve: sdk.NewDec(3 * common.TO_MICRO), // 3e6
				BaseReserve:  sdk.NewDec(1_000),               // 1e3
				SqrtDepth:    common.MustSqrtDec(sdk.NewDec(3_000 * common.TO_MICRO)),
				Config: types.MarketConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
					TradeLimitRatio:        sdk.OneDec(),
				},
			},
			shouldCreateMarket: true,
			mockIndexPrice:     sdk.OneDec().Neg(),
			oracleKeeperErr:    fmt.Errorf("No index price"),
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
			name: "market doesn't exist",
			market: types.Market{
				Pair:         asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				QuoteReserve: sdk.NewDec(3 * common.TO_MICRO), // 3e6
				BaseReserve:  sdk.NewDec(1_000),               // 1e3
				SqrtDepth:    common.MustSqrtDec(sdk.NewDec(3_000 * common.TO_MICRO)),
				Config: types.MarketConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
				},
			},
			shouldCreateMarket: false,
			err:                types.ErrPairNotSupported,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			perpammKeeper, mocks, ctx := getKeeper(t)
			ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

			if tc.shouldCreateMarket {
				assert.NoError(t, perpammKeeper.CreatePool(
					ctx,
					tc.market.Pair,
					tc.market.QuoteReserve,
					tc.market.BaseReserve,
					tc.market.Config,
					sdk.ZeroDec(),
					sdk.OneDec(),
				))
			}

			ctx = ctx.WithBlockHeight(2).WithBlockTime(time.Now().Add(5 * time.Second))

			t.Log("mock oracleKeeper index price")
			mocks.mockOracleKeeper.EXPECT().
				GetExchangeRate(ctx, tc.market.Pair).
				Return(tc.mockIndexPrice, tc.oracleKeeperErr).
				AnyTimes()

			// logged errors would be called in GetPoolPrices
			var poolPrices types.PoolPrices
			poolPrices, err := perpammKeeper.GetPoolPrices(ctx, tc.market)
			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
			} else {
				require.EqualValues(t, tc.expectedPoolPrices, poolPrices)
			}
		})
	}
}

func TestEditSwapInvariant(t *testing.T) {
	pair := asset.Registry.Pair(denoms.NIBI, denoms.NUSD)
	marketStart := types.Market{
		Pair:          pair,
		QuoteReserve:  sdk.NewDec(10 * common.TO_MICRO),
		BaseReserve:   sdk.NewDec(5 * common.TO_MICRO),
		SqrtDepth:     common.MustSqrtDec(sdk.NewDec(5 * 10 * common.TO_MICRO * common.TO_MICRO)),
		PegMultiplier: sdk.OneDec(),
		Config: types.MarketConfig{
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
		},
	}

	setupTest := func() (Keeper, sdk.Context) {
		perpammKeeper, _, ctx := getKeeper(t)
		assert.NoError(t, perpammKeeper.CreatePool(
			ctx,
			pair,
			marketStart.QuoteReserve,
			marketStart.BaseReserve,
			marketStart.Config,
			sdk.ZeroDec(),
			sdk.OneDec(),
		))
		exists := perpammKeeper.ExistsPool(ctx, pair)
		require.True(t, exists)
		return perpammKeeper, ctx
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
				Base:  marketStart.BaseReserve.MulInt64(2),
				Quote: marketStart.QuoteReserve.MulInt64(2)},
			shouldErr: false,
		},
		{
			name:                    "happy no change",
			swapInvariantMultiplier: sdk.NewDec(1),
			newReserves: Reserves{
				Base:  marketStart.BaseReserve,
				Quote: marketStart.QuoteReserve},
			shouldErr: false,
		},
		{
			name:                    "happy reserves increase 500x",
			swapInvariantMultiplier: sdk.NewDec(250_000), // 500**2
			newReserves: Reserves{
				Base:  marketStart.BaseReserve.MulInt64(500),
				Quote: marketStart.QuoteReserve.MulInt64(500)},
			shouldErr: false,
		},
		{
			name:                    "happy reserves shrink 2x",
			swapInvariantMultiplier: sdk.MustNewDecFromStr("0.25"), // (1/2)**2
			newReserves: Reserves{
				Base:  marketStart.BaseReserve.QuoInt64(2),
				Quote: marketStart.QuoteReserve.QuoInt64(2)},
			shouldErr: false,
		},
		{
			name:                    "happy reserves shrink 100x",
			swapInvariantMultiplier: sdk.MustNewDecFromStr("0.0001"), // (1/100)**2
			newReserves: Reserves{
				Base:  marketStart.BaseReserve.QuoInt64(100),
				Quote: marketStart.QuoteReserve.QuoInt64(100)},
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
			perpammKeeper, ctx := setupTest()
			if tc.shouldErr {
				err := perpammKeeper.EditSwapInvariant(ctx,
					types.EditSwapInvariantsProposal_SwapInvariantMultiple{
						Pair: pair, Multiplier: tc.swapInvariantMultiplier,
					})
				// We expect the initial config if the change fails
				assert.Error(t, err)
				market, err := perpammKeeper.Pools.Get(ctx, pair)
				assert.NoError(t, err)
				assert.EqualValues(t, marketStart.BaseReserve, market.BaseReserve)
				assert.EqualValues(t, marketStart.QuoteReserve, market.QuoteReserve)
			} else if tc.shouldPanic {
				require.Panics(t, func() {
					err := perpammKeeper.EditSwapInvariant(ctx,
						types.EditSwapInvariantsProposal_SwapInvariantMultiple{
							Pair: pair, Multiplier: tc.swapInvariantMultiplier,
						})
					require.Error(t, err)
				})
			} else {
				err := perpammKeeper.EditSwapInvariant(ctx,
					types.EditSwapInvariantsProposal_SwapInvariantMultiple{
						Pair: pair, Multiplier: tc.swapInvariantMultiplier,
					})
				// We expect the new config if the change succeeds
				require.NoError(t, err)
				market, err := perpammKeeper.Pools.Get(ctx, pair)
				assert.NoError(t, err)
				assert.EqualValues(t, tc.newReserves.Base, market.BaseReserve)
				assert.EqualValues(t, tc.newReserves.Quote, market.QuoteReserve)
			}
		})
	}
}
