package keeper

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestKeeper_getLatestCumulativePremiumFraction(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "happy path",
			test: func() {
				keeper, _, ctx := getKeeper(t)
				pair := fmt.Sprintf("%s%s%s", common.GovDenom, common.PairSeparator, common.StableDenom)

				metadata := &types.PairMetadata{
					Pair: pair,
					CumulativePremiumFractions: []sdk.Dec{
						sdk.NewDec(1),
						sdk.NewDec(2), // returns the latest from the list
					},
				}
				keeper.PairMetadata().Set(ctx, metadata)

				tokenPair, err := common.NewTokenPairFromStr(pair)
				require.NoError(t, err)
				latestCumulativePremiumFraction, err := keeper.getLatestCumulativePremiumFraction(ctx, tokenPair)
				require.NoError(t, err)

				require.Equal(t, sdk.NewDec(2), latestCumulativePremiumFraction)
			},
		},
		{
			name: "uninitialized vpool has no metadata | fail",
			test: func() {
				perpKeeper, _, ctx := getKeeper(t)
				vpool := common.TokenPair("xxx:yyy")
				lcpf, err := perpKeeper.getLatestCumulativePremiumFraction(
					ctx, vpool)
				require.Error(t, err)
				require.EqualValues(t, sdk.Dec{}, lcpf)
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

type mockedDependencies struct {
	mockAccountKeeper *mock.MockAccountKeeper
	mockBankKeeper    *mock.MockBankKeeper
	mockPriceKeeper   *mock.MockPriceKeeper
	mockVpoolKeeper   *mock.MockVpoolKeeper
}

func getKeeper(t *testing.T) (Keeper, mockedDependencies, sdk.Context) {
	storeKey := sdk.NewKVStoreKey(vpooltypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(vpooltypes.StoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	protoCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	params := initParamsKeeper(protoCodec, codec.NewLegacyAmino(), storeKey, memStoreKey)

	subSpace, found := params.GetSubspace(vpooltypes.ModuleName)
	require.True(t, found)

	ctrl := gomock.NewController(t)
	mockedAccountKeeper := mock.NewMockAccountKeeper(ctrl)
	mockedBankKeeper := mock.NewMockBankKeeper(ctrl)
	mockedPriceKeeper := mock.NewMockPriceKeeper(ctrl)
	mockedVpoolKeeper := mock.NewMockVpoolKeeper(ctrl)

	mockedAccountKeeper.
		EXPECT().GetModuleAddress(types.ModuleName).
		Return(authtypes.NewModuleAddress(vpooltypes.ModuleName))

	k := NewKeeper(
		protoCodec,
		storeKey,
		subSpace,
		mockedAccountKeeper,
		mockedBankKeeper,
		mockedPriceKeeper,
		mockedVpoolKeeper,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, nil)

	return k, mockedDependencies{
		mockAccountKeeper: mockedAccountKeeper,
		mockBankKeeper:    mockedBankKeeper,
		mockPriceKeeper:   mockedPriceKeeper,
		mockVpoolKeeper:   mockedVpoolKeeper,
	}, ctx
}

func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey sdk.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)
	paramsKeeper.Subspace(vpooltypes.ModuleName)

	return paramsKeeper
}

func TestGetPositionNotionalAndUnrealizedPnl(t *testing.T) {
	testcases := []struct {
		name string
		test func()
	}{
		{
			name: "long position; positive pnl; spot price calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_SPOT_PRICE,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(20), positionalNotional)
				require.EqualValues(t, sdk.NewDec(10), unrealizedPnl)
			},
		},
		{
			name: "long position; negative pnl; spot price calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(5), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_SPOT_PRICE,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(5), positionalNotional)
				require.EqualValues(t, sdk.NewDec(-5), unrealizedPnl)
			},
		},
		{
			name: "long position; positive pnl; twap calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(20), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_TWAP,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(20), positionalNotional)
				require.EqualValues(t, sdk.NewDec(10), unrealizedPnl)
			},
		},
		{
			name: "long position; negative pnl; twap calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(5), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_TWAP,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(5), positionalNotional)
				require.EqualValues(t, sdk.NewDec(-5), unrealizedPnl)
			},
		},
		{
			name: "long position; positive pnl; oracle calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetUnderlyingPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
					).
					Return(sdk.NewDec(2), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_ORACLE,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(20), positionalNotional)
				require.EqualValues(t, sdk.NewDec(10), unrealizedPnl)
			},
		},
		{
			name: "long position; negative pnl; oracle calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetUnderlyingPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
					).
					Return(sdk.MustNewDecFromStr("0.5"), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_ORACLE,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(5), positionalNotional)
				require.EqualValues(t, sdk.NewDec(-5), unrealizedPnl)
			},
		},
		{
			name: "short position; positive pnl; spot price calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(-10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(5), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_SPOT_PRICE,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(5), positionalNotional)
				require.EqualValues(t, sdk.NewDec(5), unrealizedPnl)
			},
		},
		{
			name: "short position; negative pnl; spot price calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(-10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_SPOT_PRICE,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(20), positionalNotional)
				require.EqualValues(t, sdk.NewDec(-10), unrealizedPnl)
			},
		},
		{
			name: "short position; positive pnl; twap calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(-10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(5), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_TWAP,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(5), positionalNotional)
				require.EqualValues(t, sdk.NewDec(5), unrealizedPnl)
			},
		},
		{
			name: "short position; negative pnl; twap calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(-10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(20), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_TWAP,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(20), positionalNotional)
				require.EqualValues(t, sdk.NewDec(-10), unrealizedPnl)
			},
		},
		{
			name: "short position; positive pnl; oracle calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(-10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetUnderlyingPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
					).
					Return(sdk.MustNewDecFromStr("0.5"), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_ORACLE,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(5), positionalNotional)
				require.EqualValues(t, sdk.NewDec(5), unrealizedPnl)
			},
		},
		{
			name: "long position; negative pnl; oracle calc",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(-10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetUnderlyingPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
					).
					Return(sdk.NewDec(2), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLCalcOption_ORACLE,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(20), positionalNotional)
				require.EqualValues(t, sdk.NewDec(-10), unrealizedPnl)
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestGetPreferencePositionNotionalAndUnrealizedPnl(t *testing.T) {
	// all tests are assumed long positions with positive pnl for ease of calculation
	// short positions and negative pnl are implicitly correct because of
	// TestGetPositionNotionalAndUnrealizedPnl
	testcases := []struct {
		name string
		test func()
	}{
		{
			name: "max pnl, pick spot price",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mock vpool spot price")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(15), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPreferencePositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLPreferenceOption_MAX,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(20), positionalNotional)
				require.EqualValues(t, sdk.NewDec(10), unrealizedPnl)
			},
		},
		{
			name: "max pnl, pick twap",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mock vpool spot price")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(30), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPreferencePositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLPreferenceOption_MAX,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(30), positionalNotional)
				require.EqualValues(t, sdk.NewDec(20), unrealizedPnl)
			},
		},
		{
			name: "min pnl, pick spot price",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mock vpool spot price")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(30), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPreferencePositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLPreferenceOption_MIN,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(20), positionalNotional)
				require.EqualValues(t, sdk.NewDec(10), unrealizedPnl)
			},
		},
		{
			name: "min pnl, pick twap",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Setting up initial position")
				oldPosition := types.Position{
					Address:      sample.AccAddress().String(),
					Pair:         "BTC:NUSD",
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}

				t.Log("Mock vpool spot price")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(15), nil)

				positionalNotional, unrealizedPnl, err := perpKeeper.
					getPreferencePositionNotionalAndUnrealizedPnL(
						ctx,
						oldPosition,
						types.PnLPreferenceOption_MIN,
					)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(15), positionalNotional)
				require.EqualValues(t, sdk.NewDec(5), unrealizedPnl)
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}
