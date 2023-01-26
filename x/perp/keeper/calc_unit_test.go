package keeper

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestCalcFreeCollateralErrors(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "invalid token pair - error",
			test: func() {
				k, _, ctx := getKeeper(t)
				alice := testutil.AccAddress()
				pos := types.ZeroPosition(ctx, common.AssetPair("foobar"), alice)
				_, err := k.calcFreeCollateral(ctx, pos)

				require.Error(t, err)
				require.ErrorIs(t, err, common.ErrInvalidTokenPair)
			},
		},
		{
			name: "token pair not found - error",
			test: func() {
				k, mocks, ctx := getKeeper(t)

				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(false)

				pos := types.ZeroPosition(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), testutil.AccAddress())

				_, err := k.calcFreeCollateral(ctx, pos)

				require.Error(t, err)
				require.ErrorIs(t, err, types.ErrPairNotFound)
			},
		},
		{
			name: "zero position",
			test: func() {
				k, mocks, ctx := getKeeper(t)

				mocks.mockVpoolKeeper.EXPECT().
					ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(true)
				mocks.mockVpoolKeeper.EXPECT().
					GetMaintenanceMarginRatio(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).
					Return(sdk.MustNewDecFromStr("0.0625"), nil)

				pos := types.ZeroPosition(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), testutil.AccAddress())

				freeCollateral, err := k.calcFreeCollateral(ctx, pos)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.ZeroDec(), freeCollateral)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestCalcFreeCollateralSuccess(t *testing.T) {
	testCases := []struct {
		name string

		positionSize           sdk.Dec
		vpoolDirection         vpooltypes.Direction
		positionNotional       sdk.Dec
		expectedFreeCollateral sdk.Dec
	}{
		{
			name:                   "long position, zero PnL",
			positionSize:           sdk.OneDec(),
			vpoolDirection:         vpooltypes.Direction_ADD_TO_POOL,
			positionNotional:       sdk.NewDec(1000),
			expectedFreeCollateral: sdk.MustNewDecFromStr("37.5"),
		},
		{
			name:                   "long position, positive PnL",
			positionSize:           sdk.OneDec(),
			vpoolDirection:         vpooltypes.Direction_ADD_TO_POOL,
			positionNotional:       sdk.NewDec(1100),
			expectedFreeCollateral: sdk.MustNewDecFromStr("31.25"),
		},
		{
			name:                   "long position, negative PnL",
			vpoolDirection:         vpooltypes.Direction_ADD_TO_POOL,
			positionSize:           sdk.OneDec(),
			positionNotional:       sdk.NewDec(970),
			expectedFreeCollateral: sdk.MustNewDecFromStr("9.375"),
		},
		{
			name:                   "long position, huge negative PnL",
			vpoolDirection:         vpooltypes.Direction_ADD_TO_POOL,
			positionSize:           sdk.OneDec(),
			positionNotional:       sdk.NewDec(900),
			expectedFreeCollateral: sdk.MustNewDecFromStr("-56.25"),
		},
		{
			name:                   "short position, zero PnL",
			positionSize:           sdk.OneDec().Neg(),
			vpoolDirection:         vpooltypes.Direction_REMOVE_FROM_POOL,
			positionNotional:       sdk.NewDec(1000),
			expectedFreeCollateral: sdk.MustNewDecFromStr("37.5"),
		},
		{
			name:                   "short position, positive PnL",
			positionSize:           sdk.OneDec().Neg(),
			vpoolDirection:         vpooltypes.Direction_REMOVE_FROM_POOL,
			positionNotional:       sdk.NewDec(900),
			expectedFreeCollateral: sdk.MustNewDecFromStr("43.75"),
		},
		{
			name:                   "short position, negative PnL",
			positionSize:           sdk.OneDec().Neg(),
			vpoolDirection:         vpooltypes.Direction_REMOVE_FROM_POOL,
			positionNotional:       sdk.NewDec(1030),
			expectedFreeCollateral: sdk.MustNewDecFromStr("5.625"),
		},
		{
			name:                   "short position, huge negative PnL",
			positionSize:           sdk.OneDec().Neg(),
			vpoolDirection:         vpooltypes.Direction_REMOVE_FROM_POOL,
			positionNotional:       sdk.NewDec(1100),
			expectedFreeCollateral: sdk.MustNewDecFromStr("-68.75"),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			k, mocks, ctx := getKeeper(t)

			pos := types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           tc.positionSize,
				Margin:                          sdk.NewDec(100),
				OpenNotional:                    sdk.NewDec(1000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			}

			t.Log("mock vpool keeper")
			mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(true)
			mocks.mockVpoolKeeper.EXPECT().
				GetMaintenanceMarginRatio(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).
				Return(sdk.MustNewDecFromStr("0.0625"), nil)
			mocks.mockVpoolKeeper.EXPECT().GetBaseAssetPrice(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				tc.vpoolDirection,
				sdk.OneDec(),
			).Return(tc.positionNotional, nil)
			mocks.mockVpoolKeeper.EXPECT().GetBaseAssetTWAP(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				tc.vpoolDirection,
				sdk.OneDec(),
				15*time.Minute,
			).Return(tc.positionNotional, nil)

			freeCollateral, err := k.calcFreeCollateral(ctx, pos)

			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedFreeCollateral, freeCollateral)
		})
	}
}

func TestGetLatestCumulativePremiumFraction(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "happy path",
			test: func() {
				keeper, _, ctx := getKeeper(t)

				metadata := &types.PairMetadata{
					Pair:                            asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
					LatestCumulativePremiumFraction: sdk.NewDec(2),
				}
				setPairMetadata(keeper, ctx, *metadata)

				latestCumulativePremiumFraction, err := keeper.
					getLatestCumulativePremiumFraction(ctx, asset.Registry.Pair(denoms.NIBI, denoms.NUSD))

				require.NoError(t, err)
				assert.Equal(t, sdk.NewDec(2), latestCumulativePremiumFraction)
			},
		},
		{
			name: "uninitialized vpool has no metadata | fail",
			test: func() {
				perpKeeper, _, ctx := getKeeper(t)
				vpool := common.AssetPair("xxx:yyy")
				lcpf, err := perpKeeper.getLatestCumulativePremiumFraction(
					ctx, vpool)
				require.Error(t, err)
				assert.EqualValues(t, sdk.Dec{}, lcpf)
			},
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}
