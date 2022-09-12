package keeper

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilmock "github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestCreatePool(t *testing.T) {
	vpoolKeeper, _, ctx := getKeeper(t)

	vpoolKeeper.CreatePool(
		ctx,
		common.PairBTCStable,
		sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
		sdk.NewDec(10_000_000),       // 10 tokens
		sdk.NewDec(5_000_000),        // 5 tokens
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
		sdk.MustNewDecFromStr("0.0625"),
		sdk.MustNewDecFromStr("15"),
	)

	exists := vpoolKeeper.ExistsPool(ctx, common.PairBTCStable)
	require.True(t, exists)

	notExist := vpoolKeeper.ExistsPool(ctx, common.AssetPair{
		Token0: "BTC",
		Token1: "OTHER",
	})
	require.False(t, notExist)
}

func TestKeeper_GetAllPools(t *testing.T) {
	vpoolKeeper, _, ctx := getKeeper(t)

	var vpools = []*types.Pool{
		{
			Pair:                   common.PairBTCStable,
			BaseAssetReserve:       sdk.NewDec(1_000_000),      // 1
			QuoteAssetReserve:      sdk.NewDec(30_000_000_000), // 30,000
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.88"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.20"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.20"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
		{
			Pair:                   common.PairETHStable,
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
				vpoolWithInvalidPair := types.Pool{
					Pair: common.AssetPair{Token0: "o:o", Token1: "unibi"}}
				vpoolKeeper, _, ctx := getKeeper(t)
				_, err := vpoolKeeper.GetPoolPrices(ctx, vpoolWithInvalidPair)
				require.ErrorContains(t, err, common.ErrInvalidTokenPair.Error())
			},
		},
		{
			name: "attempt to use vpool that hasn't been added",
			test: func(t *testing.T) {
				vpool := types.Pool{Pair: common.MustNewAssetPair("uatom:unibi")}
				vpoolKeeper, _, ctx := getKeeper(t)
				_, err := vpoolKeeper.GetPoolPrices(ctx, vpool)
				require.ErrorContains(t, err, types.ErrPairNotSupported.Error())
			},
		},
		{
			name: "vpool with reserves that don't make sense",
			test: func(t *testing.T) {
				vpool := types.Pool{
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
	validVpools := struct {
		ethstable types.Pool
		fooBar    types.Pool
		xxxyyy    types.Pool
	}{
		ethstable: types.Pool{
			Pair:                   common.PairETHStable,
			BaseAssetReserve:       sdk.NewDec(1).MulInt64(1_000),    // 1e3
			QuoteAssetReserve:      sdk.NewDec(3_000).MulInt64(1000), // 3e6
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
		fooBar: types.Pool{
			Pair:                   common.MustNewAssetPair("foo:bar"),
			BaseAssetReserve:       sdk.OneDec(),
			QuoteAssetReserve:      sdk.NewDec(2),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
		xxxyyy: types.Pool{
			Pair:                   common.MustNewAssetPair("xxx:yyy"),
			BaseAssetReserve:       sdk.NewDec(2),
			QuoteAssetReserve:      sdk.OneDec(),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.30"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.30"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	}

	var ErrPriceFeedGetCurrentPrice error = fmt.Errorf(
		"mock error on k.pricefeedKeeper.GetCurrentPrice")

	testCases := []struct {
		name          string           // test case name
		vpool         types.Pool       // vpool passed to GetPoolPrices
		vpoolInStore  bool             // whether to write 'vpool' into the kv store
		indexPriceVal sdk.Dec          // indexPriceVal returned by the x/pricefeed keepr
		twapMarkVal   sdk.Dec          // twapMarkVal returned by the x/pricefeed keepr
		err           error            // An error raised from calling Keeper.GetPoolPrices
		loggedErrs    []error          // Contains the silent errors logged on the context (ctx)
		output        types.PoolPrices // expected output from callign GetPoolPrices
	}{
		{
			name:          "happy path - vpool + prices active",
			vpool:         validVpools.ethstable,
			vpoolInStore:  true,
			indexPriceVal: sdk.NewDec(99),
			twapMarkVal:   sdk.NewDec(99),
			err:           nil,
			loggedErrs:    nil,
			output: types.PoolPrices{
				Pair:          validVpools.ethstable.Pair.String(),
				MarkPrice:     sdk.NewDec(3_000),
				IndexPrice:    sdk.NewDec(99).String(),
				TwapMark:      sdk.NewDec(99).String(),
				SwapInvariant: sdk.NewInt(3_000_000_000), // 1e3 * 3e6 = 3e9
			},
		},
		{
			name:          "happy path - vpool active + no index or twapMark",
			vpool:         validVpools.fooBar, // k = 2, markPrice = 2
			vpoolInStore:  true,
			err:           nil,
			indexPriceVal: sdk.Dec{},
			twapMarkVal:   sdk.Dec{},
			loggedErrs:    []error{ErrPriceFeedGetCurrentPrice, types.ErrNoValidTWAP},
			output: types.PoolPrices{
				Pair:          validVpools.fooBar.Pair.String(),
				MarkPrice:     sdk.NewDec(2),
				IndexPrice:    "",
				TwapMark:      "",
				SwapInvariant: sdk.NewInt(2), // 1e3 * 3e6 = 3e9
			},
		},
		{
			name:         "happy path - invalid indexPriceVal - no price feed",
			vpool:        validVpools.xxxyyy, // k = 2, markPrice = 0.5
			vpoolInStore: true,
			twapMarkVal:  sdk.NewDec(99),
			err:          nil,
			loggedErrs:   []error{ErrPriceFeedGetCurrentPrice},
			output: types.PoolPrices{
				Pair:          validVpools.xxxyyy.Pair.String(),
				MarkPrice:     sdk.MustNewDecFromStr("0.5"),
				IndexPrice:    "",
				TwapMark:      sdk.NewDec(99).String(),
				SwapInvariant: sdk.NewInt(2), // 1e3 * 3e6 = 3e9
			},
		},
		{
			name:          "happy path - invalid twapMarkVal - no mark TWAP",
			vpool:         validVpools.xxxyyy, // k = 2, markPrice = 0.5
			vpoolInStore:  true,
			indexPriceVal: sdk.NewDec(99),
			err:           nil,
			loggedErrs:    []error{types.ErrNoValidTWAP},
			output: types.PoolPrices{
				Pair:          validVpools.xxxyyy.Pair.String(),
				MarkPrice:     sdk.MustNewDecFromStr("0.5"),
				IndexPrice:    sdk.NewDec(99).String(),
				TwapMark:      "",
				SwapInvariant: sdk.NewInt(2), // 1e3 * 3e6 = 3e9
			},
		},
		{
			name:         "sadge - vpool doesn't exist",
			vpool:        validVpools.fooBar,
			vpoolInStore: false,
			err:          types.ErrPairNotSupported,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, mocks, ctx := getKeeper(t)
			ctx, mockLogger := testutilmock.AppendCtxWithMockLogger(t, ctx)

			if tc.vpoolInStore {
				vpoolKeeper.CreatePool(
					ctx, tc.vpool.Pair, tc.vpool.TradeLimitRatio, tc.vpool.QuoteAssetReserve, tc.vpool.BaseAssetReserve, tc.vpool.FluctuationLimitRatio, tc.vpool.MaxOracleSpreadRatio, tc.vpool.MaintenanceMarginRatio, tc.vpool.MaxLeverage)
			} else {
				// sanity check to make sure the test case vpool is not a genesis vpool
				prices, err := vpoolKeeper.GetPoolPrices(ctx, tc.vpool)
				require.ErrorContains(t, err, types.ErrPairNotSupported.Error())
				require.EqualValues(t, types.PoolPrices{}, prices)
			}

			// TODO indexPriceVal mock with pf keeper ?
			pair := tc.vpool.Pair
			currPrice := pftypes.CurrentPrice{
				PairID: pair.String(),
				Price:  sdk.Dec{}}
			switch {
			case (tc.indexPriceVal != sdk.Dec{}):
				currPrice.Price = tc.indexPriceVal
				mocks.mockPricefeedKeeper.EXPECT().
					GetCurrentPrice(ctx, pair.BaseDenom(), pair.QuoteDenom()).
					Return(currPrice, nil)
			case tc.err != nil:
				assert.EqualValues(t, len(tc.loggedErrs), 0)
			default:
				mockLogger.EXPECT().With("module", fmt.Sprintf("x/%s", types.ModuleName)).
					Return(ctx.Logger())
				assert.GreaterOrEqual(t, len(tc.loggedErrs), 1)
				pricefeedError := new(error)
				if tc.err != nil {
					*pricefeedError = tc.err
				} else {
					*pricefeedError = ErrPriceFeedGetCurrentPrice
				}
				mocks.mockPricefeedKeeper.EXPECT().
					GetCurrentPrice(ctx, pair.BaseDenom(), pair.QuoteDenom()).
					Return(currPrice, *pricefeedError)
			}

			// TODO twapMarkVal mock with pf keeper ?
			twap := types.CurrentTWAP{
				PairID:      pair.String(),
				Numerator:   tc.twapMarkVal,
				Denominator: sdk.OneDec(),
				Price:       tc.twapMarkVal,
			}

			numLoggedErrs := len(tc.loggedErrs)
			errMsg := fmt.Sprintf("numLoggedErrs: %v", numLoggedErrs)
			switch {
			case (tc.twapMarkVal != sdk.Dec{}):
				var twapKey []byte = types.CurrentTWAPKey(pair)
				ctx.KVStore(vpoolKeeper.storeKey).Set(twapKey, vpoolKeeper.codec.MustMarshal(&twap))
				if tc.err != nil {
					assert.EqualValues(t, len(tc.loggedErrs), 0)
				} else {
					assert.Truef(t, (numLoggedErrs == 0) || (numLoggedErrs == 1), errMsg)
				}
			case tc.err != nil:
				assert.Truef(t, (numLoggedErrs == 0) || (numLoggedErrs == 1), errMsg)
			default:
				mockLogger.EXPECT().With("module", fmt.Sprintf("x/%s", types.ModuleName)).
					Return(ctx.Logger())
			}

			t.Log("Call EXPECT for all errors expected to be logged on the ctx ")
			if len(tc.loggedErrs) > 0 {
				for _, loggedErr := range tc.loggedErrs {
					mockLogger.EXPECT().Error(loggedErr.Error())
				}
			}

			// logged errors would be called in GetPoolPrices
			var poolPrices types.PoolPrices
			poolPrices, err := vpoolKeeper.GetPoolPrices(ctx, tc.vpool)
			if tc.err != nil {
				assert.ErrorContains(t, err, tc.err.Error())
			} else {
				assert.EqualValues(t, poolPrices.IndexPrice, tc.output.IndexPrice)
				assert.EqualValues(t, poolPrices.TwapMark, tc.output.TwapMark)
				assert.EqualValues(t, poolPrices.Pair, tc.output.Pair)
				assert.EqualValues(t, poolPrices.MarkPrice, tc.output.MarkPrice)
				assert.EqualValues(t, poolPrices.SwapInvariant, tc.output.SwapInvariant)
				assert.EqualValues(t, poolPrices.BlockNumber, ctx.BlockHeight())
			}
		})
	}
}
