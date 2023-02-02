package keeper

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestSwapQuoteForBase(t *testing.T) {
	tests := []struct {
		name                      string
		pair                      common.AssetPair
		direction                 types.Direction
		quoteAmount               sdk.Dec
		baseLimit                 sdk.Dec
		skipFluctuationLimitCheck bool

		expectedQuoteReserve sdk.Dec
		expectedBaseReserve  sdk.Dec
		expectedBaseAmount   sdk.Dec
		expectedErr          error
	}{
		{
			name:                      "quote amount == 0",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			quoteAmount:               sdk.NewDec(0),
			baseLimit:                 sdk.NewDec(10),
			skipFluctuationLimitCheck: false,

			expectedQuoteReserve: sdk.NewDec(10 * common.Precision),
			expectedBaseReserve:  sdk.NewDec(5 * common.Precision),
			expectedBaseAmount:   sdk.ZeroDec(),
		},
		{
			name:                      "normal swap add",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			quoteAmount:               sdk.NewDec(100_000),
			baseLimit:                 sdk.NewDec(49504),
			skipFluctuationLimitCheck: false,

			expectedQuoteReserve: sdk.NewDec(10_100_000),
			expectedBaseReserve:  sdk.MustNewDecFromStr("4950495.049504950495049505"),
			expectedBaseAmount:   sdk.MustNewDecFromStr("49504.950495049504950495"),
		},
		{
			name:                      "normal swap remove",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			quoteAmount:               sdk.NewDec(100_000),
			baseLimit:                 sdk.NewDec(50506),
			skipFluctuationLimitCheck: false,

			expectedQuoteReserve: sdk.NewDec(9_900_000),
			expectedBaseReserve:  sdk.MustNewDecFromStr("5050505.050505050505050505"),
			expectedBaseAmount:   sdk.MustNewDecFromStr("50505.050505050505050505"),
		},
		{
			name:                      "pair not supported",
			pair:                      "abc:xyz",
			direction:                 types.Direction_ADD_TO_POOL,
			quoteAmount:               sdk.NewDec(10),
			baseLimit:                 sdk.NewDec(10),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrPairNotSupported,
		},
		{
			name:                      "base amount less than base limit in Long",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			quoteAmount:               sdk.NewDec(500_000),
			baseLimit:                 sdk.NewDec(454_500),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrAssetFailsUserLimit,
		},
		{
			name:                      "base amount more than base limit in Short",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			quoteAmount:               sdk.NewDec(1 * common.Precision),
			baseLimit:                 sdk.NewDec(454_500),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrAssetFailsUserLimit,
		},
		{
			name:                      "over trading limit when removing quote",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			quoteAmount:               sdk.NewDec(9_000_001),
			baseLimit:                 sdk.ZeroDec(),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrOverTradingLimit,
		},
		{
			name:                      "over trading limit when adding quote",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			quoteAmount:               sdk.NewDec(9_000_001),
			baseLimit:                 sdk.ZeroDec(),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrOverTradingLimit,
		},
		{
			name:                      "over fluctuation limit fails on add",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			quoteAmount:               sdk.NewDec(1 * common.Precision),
			baseLimit:                 sdk.NewDec(454_544),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrOverFluctuationLimit,
		},
		{
			name:                      "over fluctuation limit fails on remove",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			quoteAmount:               sdk.NewDec(1 * common.Precision),
			baseLimit:                 sdk.NewDec(555_556),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrOverFluctuationLimit,
		},
		{
			name:                      "over fluctuation limit allowed on add",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			quoteAmount:               sdk.NewDec(1 * common.Precision),
			baseLimit:                 sdk.NewDec(454_544),
			skipFluctuationLimitCheck: true,

			expectedQuoteReserve: sdk.NewDec(11 * common.Precision),
			expectedBaseReserve:  sdk.MustNewDecFromStr("4545454.545454545454545455"),
			expectedBaseAmount:   sdk.MustNewDecFromStr("454545.454545454545454545"),
		},
		{
			name:                      "over fluctuation limit allowed on remove",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			quoteAmount:               sdk.NewDec(1 * common.Precision),
			baseLimit:                 sdk.NewDec(555_556),
			skipFluctuationLimitCheck: true,

			expectedQuoteReserve: sdk.NewDec(9 * common.Precision),
			expectedBaseReserve:  sdk.MustNewDecFromStr("5555555.555555555555555556"),
			expectedBaseAmount:   sdk.MustNewDecFromStr("555555.555555555555555556"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			oracleKeeper := mock.NewMockOracleKeeper(gomock.NewController(t))
			vpoolKeeper, ctx := VpoolKeeper(t, oracleKeeper)

			oracleKeeper.EXPECT().GetExchangeRate(gomock.Any(), gomock.Any()).Return(sdk.NewDec(1), nil).AnyTimes()

			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* quoteAssetReserve */ sdk.NewDec(10*common.Precision), // 10 tokens
				/* baseAssetReserve */ sdk.NewDec(5*common.Precision), // 5 tokens
				types.VpoolConfig{
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))

			baseAmt, err := vpoolKeeper.SwapQuoteForBase(
				ctx,
				tc.pair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
				tc.skipFluctuationLimitCheck,
			)

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				assert.EqualValuesf(t, tc.expectedBaseAmount, baseAmt, "base amount mismatch")

				t.Log("assert vpool")
				pool, err := vpoolKeeper.Pools.Get(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
				require.NoError(t, err)
				assert.EqualValuesf(t, tc.expectedQuoteReserve, pool.QuoteAssetReserve, "pool quote asset reserve mismatch")
				assert.EqualValuesf(t, tc.expectedBaseReserve, pool.BaseAssetReserve, "pool base asset reserve mismatch")
			}
		})
	}
}

func TestSwapBaseForQuote(t *testing.T) {
	tests := []struct {
		name                      string
		pair                      common.AssetPair
		direction                 types.Direction
		baseAmt                   sdk.Dec
		quoteLimit                sdk.Dec
		skipFluctuationLimitCheck bool

		expectedQuoteReserve     sdk.Dec
		expectedBaseReserve      sdk.Dec
		expectedQuoteAssetAmount sdk.Dec
		expectedErr              error
	}{
		{
			name:                      "zero base asset swap",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			baseAmt:                   sdk.ZeroDec(),
			quoteLimit:                sdk.ZeroDec(),
			skipFluctuationLimitCheck: false,

			expectedQuoteReserve:     sdk.NewDec(10 * common.Precision),
			expectedBaseReserve:      sdk.NewDec(5 * common.Precision),
			expectedQuoteAssetAmount: sdk.ZeroDec(),
		},
		{
			name:                      "add base asset swap",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			baseAmt:                   sdk.NewDec(100_000),
			quoteLimit:                sdk.NewDec(196078),
			skipFluctuationLimitCheck: false,

			expectedQuoteReserve:     sdk.MustNewDecFromStr("9803921.568627450980392157"),
			expectedBaseReserve:      sdk.NewDec(5_100_000),
			expectedQuoteAssetAmount: sdk.MustNewDecFromStr("196078.431372549019607843"),
		},
		{
			name:                      "remove base asset",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			baseAmt:                   sdk.NewDec(100_000),
			quoteLimit:                sdk.NewDec(204_082),
			skipFluctuationLimitCheck: false,

			expectedQuoteReserve:     sdk.MustNewDecFromStr("10204081.632653061224489796"),
			expectedBaseReserve:      sdk.NewDec(4_900_000),
			expectedQuoteAssetAmount: sdk.MustNewDecFromStr("204081.632653061224489796"),
		},
		{
			name:                      "pair not supported",
			pair:                      "abc:xyz",
			direction:                 types.Direction_ADD_TO_POOL,
			baseAmt:                   sdk.NewDec(10),
			quoteLimit:                sdk.NewDec(10),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrPairNotSupported,
		},
		{
			name:                      "quote amount less than quote limit in Long",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			baseAmt:                   sdk.NewDec(100_000),
			quoteLimit:                sdk.NewDec(196079),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrAssetFailsUserLimit,
		},
		{
			name:                      "quote amount more than quote limit in Short",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			baseAmt:                   sdk.NewDec(100_000),
			quoteLimit:                sdk.NewDec(204_081),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrAssetFailsUserLimit,
		},
		{
			name:                      "over trading limit when removing base",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			baseAmt:                   sdk.NewDec(4_500_001),
			quoteLimit:                sdk.ZeroDec(),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrOverTradingLimit,
		},
		{
			name:                      "over trading limit when adding base",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			baseAmt:                   sdk.NewDec(4_500_001),
			quoteLimit:                sdk.ZeroDec(),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrOverTradingLimit,
		},
		{
			name:                      "over fluctuation limit fails on add",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			baseAmt:                   sdk.NewDec(1 * common.Precision),
			quoteLimit:                sdk.NewDec(1_666_666),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrOverFluctuationLimit,
		},
		{
			name:                      "over fluctuation limit fails on remove",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			baseAmt:                   sdk.NewDec(1 * common.Precision),
			quoteLimit:                sdk.NewDec(2_500_001),
			skipFluctuationLimitCheck: false,

			expectedErr: types.ErrOverFluctuationLimit,
		},
		{
			name:                      "over fluctuation limit allowed on add",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_ADD_TO_POOL,
			baseAmt:                   sdk.NewDec(1 * common.Precision),
			quoteLimit:                sdk.NewDec(1_666_666),
			skipFluctuationLimitCheck: true,

			expectedQuoteReserve:     sdk.MustNewDecFromStr("8333333.333333333333333333"),
			expectedBaseReserve:      sdk.NewDec(6 * common.Precision),
			expectedQuoteAssetAmount: sdk.MustNewDecFromStr("1666666.666666666666666667"),
		},
		{
			name:                      "over fluctuation limit allowed on remove",
			pair:                      asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			direction:                 types.Direction_REMOVE_FROM_POOL,
			baseAmt:                   sdk.NewDec(1 * common.Precision),
			quoteLimit:                sdk.NewDec(2_500_001),
			skipFluctuationLimitCheck: true,

			expectedQuoteReserve:     sdk.NewDec(12_500_000),
			expectedBaseReserve:      sdk.NewDec(4 * common.Precision),
			expectedQuoteAssetAmount: sdk.NewDec(2_500_000),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pfKeeper := mock.NewMockOracleKeeper(gomock.NewController(t))

			vpoolKeeper, ctx := VpoolKeeper(t, pfKeeper)
			pfKeeper.EXPECT().GetExchangeRate(gomock.Any(), gomock.Any()).Return(sdk.NewDec(1), nil).AnyTimes()

			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* quoteAssetReserve */ sdk.NewDec(10*common.Precision), // 10 tokens
				/* baseAssetReserve */ sdk.NewDec(5*common.Precision), // 5 tokens
				types.VpoolConfig{
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))

			quoteAssetAmount, err := vpoolKeeper.SwapBaseForQuote(
				ctx,
				tc.pair,
				tc.direction,
				tc.baseAmt,
				tc.quoteLimit,
				tc.skipFluctuationLimitCheck,
			)

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.EqualValuesf(t, tc.expectedQuoteAssetAmount, quoteAssetAmount,
					"expected %s; got %s", tc.expectedQuoteAssetAmount.String(), quoteAssetAmount.String())

				t.Log("assert pool")
				pool, err := vpoolKeeper.Pools.Get(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
				require.NoError(t, err)
				assert.Equal(t, tc.expectedQuoteReserve, pool.QuoteAssetReserve)
				assert.Equal(t, tc.expectedBaseReserve, pool.BaseAssetReserve)
			}
		})
	}
}

func TestGetVpools(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockOracleKeeper(gomock.NewController(t)),
	)

	assert.NoError(t, vpoolKeeper.CreatePool(
		ctx,
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		sdk.NewDec(10*common.Precision),
		sdk.NewDec(5*common.Precision),
		types.VpoolConfig{
			TradeLimitRatio:        sdk.OneDec(),
			FluctuationLimitRatio:  sdk.OneDec(),
			MaxOracleSpreadRatio:   sdk.OneDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	))
	assert.NoError(t, vpoolKeeper.CreatePool(
		ctx,
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		sdk.NewDec(5*common.Precision),
		sdk.NewDec(10*common.Precision),
		types.VpoolConfig{
			TradeLimitRatio:        sdk.OneDec(),
			FluctuationLimitRatio:  sdk.OneDec(),
			MaxOracleSpreadRatio:   sdk.OneDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	))

	pools := vpoolKeeper.Pools.Iterate(ctx, collections.Range[common.AssetPair]{}).Values()

	require.EqualValues(t, 2, len(pools))

	require.EqualValues(t, pools[0], types.Vpool{
		Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		BaseAssetReserve:  sdk.NewDec(5 * common.Precision),
		QuoteAssetReserve: sdk.NewDec(10 * common.Precision),
		Config: types.VpoolConfig{
			TradeLimitRatio:        sdk.OneDec(),
			FluctuationLimitRatio:  sdk.OneDec(),
			MaxOracleSpreadRatio:   sdk.OneDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	})
	require.EqualValues(t, pools[1], types.Vpool{
		Pair:              asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		BaseAssetReserve:  sdk.NewDec(10 * common.Precision),
		QuoteAssetReserve: sdk.NewDec(5 * common.Precision),
		Config: types.VpoolConfig{
			TradeLimitRatio:        sdk.OneDec(),
			FluctuationLimitRatio:  sdk.OneDec(),
			MaxOracleSpreadRatio:   sdk.OneDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	})
}

func TestCheckFluctuationLimitRatio(t *testing.T) {
	tests := []struct {
		name              string
		pool              types.Vpool
		existingSnapshots []types.ReserveSnapshot

		expectedErr error
	}{
		{
			name: "uses latest snapshot - does not result in error",
			pool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				QuoteAssetReserve: sdk.NewDec(1002),
				BaseAssetReserve:  sdk.OneDec(),
				Config: types.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.001"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			},
			existingSnapshots: []types.ReserveSnapshot{
				{
					Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					QuoteAssetReserve: sdk.NewDec(1000),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       0,
				},
				{
					Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					QuoteAssetReserve: sdk.NewDec(1002),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       1,
				},
			},
			expectedErr: nil,
		},
		{
			name: "uses previous snapshot - results in error",
			pool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				QuoteAssetReserve: sdk.NewDec(1002),
				BaseAssetReserve:  sdk.OneDec(),
				Config: types.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.001"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			},
			existingSnapshots: []types.ReserveSnapshot{
				{
					Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					QuoteAssetReserve: sdk.NewDec(1000),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       0,
				},
			},
			expectedErr: types.ErrOverFluctuationLimit,
		},
		{
			name: "only one snapshot - no error",
			pool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				QuoteAssetReserve: sdk.NewDec(1000),
				BaseAssetReserve:  sdk.OneDec(),
				Config: types.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.001"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			},
			existingSnapshots: []types.ReserveSnapshot{
				{
					Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					QuoteAssetReserve: sdk.NewDec(1000),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       0,
				},
			},
			expectedErr: nil,
		},
		{
			name: "zero fluctuation limit - no error",
			pool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				QuoteAssetReserve: sdk.NewDec(2000),
				BaseAssetReserve:  sdk.OneDec(),
				Config: types.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.ZeroDec(),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			},
			existingSnapshots: []types.ReserveSnapshot{
				{
					Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					QuoteAssetReserve: sdk.NewDec(1000),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       0,
				},
				{
					Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					QuoteAssetReserve: sdk.NewDec(1002),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       1,
				},
			},
			expectedErr: nil,
		},
		{
			name: "multiple pools - no overlap",
			pool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				QuoteAssetReserve: sdk.NewDec(1000),
				BaseAssetReserve:  sdk.OneDec(),
				Config: types.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.001"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			},
			existingSnapshots: []types.ReserveSnapshot{
				{
					Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					QuoteAssetReserve: sdk.NewDec(1000),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       0,
				},
				{
					Pair:              asset.Registry.Pair(denoms.ETH, denoms.NUSD),
					QuoteAssetReserve: sdk.NewDec(2000),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       0,
				},
				{
					Pair:              asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
					QuoteAssetReserve: sdk.NewDec(2000),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       0,
				},
				{
					Pair:              asset.Registry.Pair(denoms.USDC, denoms.NUSD),
					QuoteAssetReserve: sdk.OneDec(),
					BaseAssetReserve:  sdk.OneDec(),
					TimestampMs:       0,
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)),
			)

			vpoolKeeper.Pools.Insert(ctx, tc.pool.Pair, tc.pool)

			for _, snapshot := range tc.existingSnapshots {
				vpoolKeeper.ReserveSnapshots.Insert(
					ctx,
					collections.Join(
						snapshot.Pair,
						time.UnixMilli(snapshot.TimestampMs)),
					snapshot)
			}

			t.Log("check fluctuation limit")
			err := vpoolKeeper.checkFluctuationLimitRatio(ctx, tc.pool)

			t.Log("check error if any")
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetMaintenanceMarginRatio(t *testing.T) {
	tests := []struct {
		name string
		pool types.Vpool

		expectedMaintenanceMarginRatio sdk.Dec
	}{
		{
			name: "zero fluctuation limit ratio",
			pool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				QuoteAssetReserve: sdk.OneDec(),
				BaseAssetReserve:  sdk.OneDec(),
				Config: types.DefaultVpoolConfig().
					WithMaintenanceMarginRatio(sdk.MustNewDecFromStr("0.9876")),
			},
			expectedMaintenanceMarginRatio: sdk.MustNewDecFromStr("0.9876"),
		},
		{
			name: "zero fluctuation limit ratio",
			pool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				QuoteAssetReserve: sdk.OneDec(),
				BaseAssetReserve:  sdk.OneDec(),
				Config: types.DefaultVpoolConfig().
					WithMaintenanceMarginRatio(sdk.MustNewDecFromStr("0.4242")),
			},
			expectedMaintenanceMarginRatio: sdk.MustNewDecFromStr("0.4242"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)),
			)
			vpoolKeeper.Pools.Insert(ctx, tc.pool.Pair, tc.pool)
			mmr, err := vpoolKeeper.GetMaintenanceMarginRatio(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
			assert.NoError(t, err)
			assert.EqualValues(t, tc.expectedMaintenanceMarginRatio, mmr)
		})
	}
}

func TestGetMaxLeverage(t *testing.T) {
	tests := []struct {
		name string
		pool types.Vpool

		expectedMaxLeverage sdk.Dec
	}{
		{
			name: "zero fluctuation limit ratio",
			pool: types.Vpool{
				Pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				QuoteAssetReserve: sdk.OneDec(),
				BaseAssetReserve:  sdk.OneDec(),
				Config: types.VpoolConfig{
					FluctuationLimitRatio:  sdk.ZeroDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.42"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					TradeLimitRatio:        sdk.OneDec(),
				},
			},
			expectedMaxLeverage: sdk.MustNewDecFromStr("15"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)),
			)
			vpoolKeeper.Pools.Insert(ctx, tc.pool.Pair, tc.pool)

			maxLeverage, err := vpoolKeeper.GetMaxLeverage(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
			assert.EqualValues(t, tc.expectedMaxLeverage, maxLeverage)
			assert.NoError(t, err)
		})
	}
}
