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
	tests := []struct {
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

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestSwapQuoteAssetForBase(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "long position - buy",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAmount=*/ sdk.NewDec(10),
						/*baseLimit=*/ sdk.NewDec(1),
					).Return(sdk.NewDec(5), nil)

				baseAmount, err := perpKeeper.swapQuoteForBase(
					ctx,
					common.TokenPair("BTC:NUSD"),
					types.Side_BUY,
					sdk.NewDec(10),
					sdk.NewDec(1),
					false,
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(5), baseAmount)
			},
		},
		{
			name: "short position - sell",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAmount=*/ sdk.NewDec(10),
						/*baseLimit=*/ sdk.NewDec(1),
					).Return(sdk.NewDec(5), nil)

				baseAmount, err := perpKeeper.swapQuoteForBase(
					ctx,
					common.TokenPair("BTC:NUSD"),
					types.Side_SELL,
					sdk.NewDec(10),
					sdk.NewDec(1),
					false,
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(-5), baseAmount)
			},
		},
	}

	for _, tc := range tests {
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

func TestIncreasePosition(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "increase long position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1BTC=1NUSD)
			// BTC went up in value, now its price is 1BTC=2NUSD
			// user increases position by another 10 NUSD at 10x leverage
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(10),  // 10 NUSD
					OpenNotional:                        sdk.NewDec(100), // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(50),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(50), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				resp, err := perpKeeper.increasePosition(
					ctx,
					currentPosition,
					types.Side_BUY,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(50), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount)
				require.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				require.EqualValues(t, sdk.NewDec(50), resp.ExchangedPositionSize)
				require.EqualValues(t, sdk.NewDec(2), resp.FundingPayment)
				require.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)
				require.EqualValues(t, sdk.NewDec(10), resp.MarginToVault)
				require.EqualValues(t, sdk.NewDec(100), resp.UnrealizedPnlAfter)

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.NewDec(150), resp.Position.Size_)        // 100 + 50
				require.EqualValues(t, sdk.NewDec(18), resp.Position.Margin)        // 10(old) + 10(new) - 2(funding payment)
				require.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "increase long position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1BTC=1NUSD)
			// BTC went down in value, now its price is 1.01BTC=1NUSD
			// user increases position by another 10 NUSD at 10x leverage
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(10),  // 10 NUSD
					OpenNotional:                        sdk.NewDec(100), // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(101),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(101), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(99), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				resp, err := perpKeeper.increasePosition(
					ctx,
					currentPosition,
					types.Side_BUY,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(101), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // equal to open notional
				require.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				require.EqualValues(t, sdk.NewDec(101), resp.ExchangedPositionSize) // equal to base amount bought
				require.EqualValues(t, sdk.NewDec(2), resp.FundingPayment)          // 0.02 * 100
				require.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)             // always zero for increasePosition
				require.EqualValues(t, sdk.NewDec(10), resp.MarginToVault)          // openNotional / leverage
				require.EqualValues(t, sdk.NewDec(-1), resp.UnrealizedPnlAfter)     // 99 - 100

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.NewDec(201), resp.Position.Size_)        // 100 + 101
				require.EqualValues(t, sdk.NewDec(18), resp.Position.Margin)        // 10(old) + 10(new) - 2(funding payment)
				require.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "increase long position, bad debt due to huge funding payment",
			// user bought in at 110 BTC for 11 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// open and positional notional value is 110 NUSD
			// BTC went down in value, now its price is 1.1 BTC = 1 NUSD
			// position notional value is 100 NUSD, unrealized PnL is -10 NUSD
			// user increases position by another 10 NUSD at 10x leverage
			// funding payment causes negative margin aka bad debt
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(110), // 110 BTC
					Margin:                              sdk.NewDec(11),  // 11 NUSD
					OpenNotional:                        sdk.NewDec(110), // 110 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(110),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(110), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(110),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.2"), // 0.2 NUSD / BTC
					},
				})

				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				resp, err := perpKeeper.increasePosition(
					ctx,
					currentPosition,
					types.Side_BUY,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(110), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(10), resp.MarginToVault) // openNotional / leverage
				require.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)    // always zero for increasePosition

				require.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount)  // equal to open notional
				require.EqualValues(t, sdk.NewDec(110), resp.ExchangedPositionSize)      // equal to base amount bought
				require.EqualValues(t, sdk.MustNewDecFromStr("22"), resp.FundingPayment) // 0.02 * 110
				require.EqualValues(t, sdk.NewDec(-10), resp.UnrealizedPnlAfter)         // 90 - 100
				require.EqualValues(t, sdk.NewDec(1), resp.BadDebt)                      // 11(old) + 10(new) - 22(funding payment)

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.NewDec(220), resp.Position.Size_)        // 110 + 110
				require.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)         // 11(old) + 10(new) - 22(funding payment) --> zero margin left
				require.EqualValues(t, sdk.NewDec(210), resp.Position.OpenNotional) // 100(old) + 100(new)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.2"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "increase short position, positive PnL",
			// user sold 100 BTC for 100 NUSD at 10x leverage (1BTC=1NUSD)
			// user's initial margin deposit was 10 NUSD
			// BTC went down in value, now its price is 2BTC=1NUSD
			// user increases position by another 10 NUSD at 10x leverage
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-100), // -100 BTC
					Margin:                              sdk.NewDec(10),   // 10 NUSD
					OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(200),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(50), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				resp, err := perpKeeper.increasePosition(
					ctx,
					currentPosition,
					types.Side_SELL,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(200), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // equal to open notional
				require.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				require.EqualValues(t, sdk.NewDec(-200), resp.ExchangedPositionSize) // equal to amount of base asset IOUs
				require.EqualValues(t, sdk.NewDec(-2), resp.FundingPayment)          // -100 * 0.02
				require.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)              // always zero for increasePosition
				require.EqualValues(t, sdk.NewDec(10), resp.MarginToVault)           // open notional / leverage
				require.EqualValues(t, sdk.NewDec(50), resp.UnrealizedPnlAfter)      // 100 - 50

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.NewDec(-300), resp.Position.Size_)       // -100 - 200
				require.EqualValues(t, sdk.NewDec(22), resp.Position.Margin)        // 10(old) + 10(new)  - (-2)(funding payment)
				require.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "increase short position, negative PnL",
			// user sold 100 BTC for 100 NUSD at 10x leverage (1BTC=1NUSD)
			// user's initial margin deposit was 10 NUSD
			// BTC went up in value, now its price is 0.99BTC=1NUSD
			// user increases position by another 10 NUSD at 10x leverage
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-100), // 100 BTC
					Margin:                              sdk.NewDec(10),   // 10 NUSD
					OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(99),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(99), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(101), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				resp, err := perpKeeper.increasePosition(
					ctx,
					currentPosition,
					types.Side_SELL,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(99), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // equal to open notional
				require.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				require.EqualValues(t, sdk.NewDec(-99), resp.ExchangedPositionSize) // base asset IOUs
				require.EqualValues(t, sdk.NewDec(-2), resp.FundingPayment)         // -100 * 0.02
				require.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)             // always zero for increasePosition
				require.EqualValues(t, sdk.NewDec(10), resp.MarginToVault)          // openNotional / leverage
				require.EqualValues(t, sdk.NewDec(-1), resp.UnrealizedPnlAfter)     // 100 - 101

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.NewDec(-199), resp.Position.Size_)       // -100 - 99
				require.EqualValues(t, sdk.NewDec(22), resp.Position.Margin)        // 10(old) + 10(new) - (-2)(funding payment)
				require.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "increase short position, bad debt due to huge funding payment",
			// user sold 100 BTC for 100 NUSD at 10x leverage (1BTC=1NUSD)
			// user's initial margin deposit was 10 NUSD
			// position and open notional is 100 NUSD
			// BTC went up in value, now its price is 1 BTC = 1.05 NUSD
			// position notional is 105 NUSD and unrealizedPnL is -5 NUSD
			// user increases position by another 105 NUSD at 10x leverage
			// funding payment causes bad debt
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-100), // 100 BTC
					Margin:                              sdk.NewDec(10),   // 10 NUSD
					OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(105),
						/*baseAssetLimit=*/ sdk.NewDec(100),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(105), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("-0.3"), // - 0.3 NUSD / BTC
					},
				})

				t.Log("Increase position with 10.5 NUSD margin and 10x leverage.")
				resp, err := perpKeeper.increasePosition(
					ctx,
					currentPosition,
					types.Side_SELL,
					/*openNotional=*/ sdk.NewDec(105), // NUSD
					/*baseLimit=*/ sdk.NewDec(100), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)                   // always zero for increasePosition
				require.EqualValues(t, sdk.MustNewDecFromStr("10.5"), resp.MarginToVault) // openNotional / leverage

				require.EqualValues(t, sdk.NewDec(105), resp.ExchangedQuoteAssetAmount) // equal to open notional
				require.EqualValues(t, sdk.NewDec(-100), resp.ExchangedPositionSize)    // base asset IOUs
				require.EqualValues(t, sdk.NewDec(30), resp.FundingPayment)             // -100 * (-0.2)
				require.EqualValues(t, sdk.NewDec(-5), resp.UnrealizedPnlAfter)         // 100 - 105
				require.EqualValues(t, sdk.MustNewDecFromStr("9.5"), resp.BadDebt)      // 10(old) + 10.5(new) - (30)(funding payment)

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.NewDec(-200), resp.Position.Size_)       // -100 + (-100)
				require.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)         // 10(old) + 10.5(new) - (30)(funding payment) --> zero margin left
				require.EqualValues(t, sdk.NewDec(205), resp.Position.OpenNotional) // 100(old) + 105(new)
				require.EqualValues(t, sdk.MustNewDecFromStr("-0.3"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestDecreasePosition(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "decrease long position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 0.5 BTC = 1 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user decreases position by notional value of 100 NUSD
			// user ends up with realized PnL of 50 NUSD, unrealized PnL of +50 NUSD
			//   position notional value of 100 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(10),  // 10 NUSD
					OpenNotional:                        sdk.NewDec(100), // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(50),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(50), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("decrease position by 100 NUSD in notional value")
				resp, err := perpKeeper.decreasePosition(
					ctx,
					currentPosition,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(50), // BTC
					/*canOverFluctuationLimit=*/ false,
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // open notional
				require.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				require.EqualValues(t, sdk.NewDec(-50), resp.ExchangedPositionSize) // sold back to vpool
				require.EqualValues(t, sdk.NewDec(2), resp.FundingPayment)
				require.EqualValues(t, sdk.ZeroDec(), resp.MarginToVault)
				require.EqualValues(t, sdk.NewDec(50), resp.RealizedPnl)
				require.EqualValues(t, sdk.NewDec(50), resp.UnrealizedPnlAfter)

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.NewDec(50), resp.Position.Size_)        // 100 - 50
				require.EqualValues(t, sdk.NewDec(58), resp.Position.Margin)       // 10(old) + 50(realized PnL) - 2(funding payment)
				require.EqualValues(t, sdk.NewDec(50), resp.Position.OpenNotional) // 200(position notional) - 100(notional sold) - 50(unrealized PnL)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "decrease long position, negative PnL",
			// user bought in at 105 BTC for 10.5 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 105 NUSD
			// BTC drops in value, now its price is 1.05 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -5 NUSD
			// user decreases position by notional value of 5 NUSD
			// user ends up with realized PnL of -0.25 NUSD, unrealized PnL of -4.75 NUSD,
			//   position notional value of 95 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(105),               // 105 BTC
					Margin:                              sdk.MustNewDecFromStr("10.5"), // 10.5 NUSD
					OpenNotional:                        sdk.NewDec(105),               // 105 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(105),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(5),
						/*baseAssetLimit=*/ sdk.MustNewDecFromStr("5.25"),
					).Return( /*baseAssetAmount=*/ sdk.MustNewDecFromStr("5.25"), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("decrease position by 5 NUSD in notional value")
				resp, err := perpKeeper.decreasePosition(
					ctx,
					currentPosition,
					/*openNotional=*/ sdk.NewDec(5), // NUSD
					/*baseLimit=*/ sdk.MustNewDecFromStr("5.25"), // BTC
					/*canOverFluctuationLimit=*/ false,
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(5), resp.ExchangedQuoteAssetAmount) // open notional
				require.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				require.EqualValues(t, sdk.MustNewDecFromStr("-5.25"), resp.ExchangedPositionSize) // sold back to vpool
				require.EqualValues(t, sdk.MustNewDecFromStr("2.1"), resp.FundingPayment)          // 105 * 0.02
				require.EqualValues(t, sdk.MustNewDecFromStr("-0.25"), resp.RealizedPnl)           // (-5)(unrealizedPnL) * 5.25/105 (fraction of position size reduced)
				require.EqualValues(t, sdk.MustNewDecFromStr("-4.75"), resp.UnrealizedPnlAfter)    // (-5)(unrealizedPnL) - (-0.25)(realizedPnL)
				require.EqualValues(t, sdk.ZeroDec(), resp.MarginToVault)                          // always zero for decreasePosition

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.MustNewDecFromStr("99.75"), resp.Position.Size_)        // 105 - 5.25
				require.EqualValues(t, sdk.MustNewDecFromStr("8.15"), resp.Position.Margin)        // 10(old) + (-0.25)(realized PnL) - 2.1(funding payment)
				require.EqualValues(t, sdk.MustNewDecFromStr("99.75"), resp.Position.OpenNotional) // 100(position notional) - 5(notional sold) - (-4.75)(unrealized PnL)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},

		/*==========================SHORT POSITIONS===========================*/

		{
			name: "decrease short position, positive PnL",
			// user bought in at 105 BTC for 10.5 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 105 NUSD
			// BTC drops in value, now its price is 1.05 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of 5 NUSD
			// user decreases position by notional value of 5 NUSD
			// user ends up with realized PnL of 0.25 NUSD, unrealized PnL of 4.75 NUSD,
			//   position notional value of 95 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-105),              // -105 BTC
					Margin:                              sdk.MustNewDecFromStr("10.5"), // 10.5 NUSD
					OpenNotional:                        sdk.NewDec(105),               // 105 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(105),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(5),
						/*baseAssetLimit=*/ sdk.MustNewDecFromStr("5.25"),
					).Return( /*baseAssetAmount=*/ sdk.MustNewDecFromStr("5.25"), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("decrease position by 5 NUSD in notional value")
				resp, err := perpKeeper.decreasePosition(
					ctx,
					currentPosition,
					/*openNotional=*/ sdk.NewDec(5), // NUSD
					/*baseLimit=*/ sdk.MustNewDecFromStr("5.25"), // BTC
					/*canOverFluctuationLimit=*/ false,
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.NewDec(5), resp.ExchangedQuoteAssetAmount) // open notional
				require.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				require.EqualValues(t, sdk.MustNewDecFromStr("5.25"), resp.ExchangedPositionSize) // bought back from vpool
				require.EqualValues(t, sdk.MustNewDecFromStr("-2.1"), resp.FundingPayment)        // -105 * 0.02
				require.EqualValues(t, sdk.MustNewDecFromStr("0.25"), resp.RealizedPnl)           // (-5)(unrealizedPnL) * 5.25/105 (fraction of position size reduced)
				require.EqualValues(t, sdk.MustNewDecFromStr("4.75"), resp.UnrealizedPnlAfter)    // (-5)(unrealizedPnL) - (-0.25)(realizedPnL)
				require.EqualValues(t, sdk.ZeroDec(), resp.MarginToVault)                         // always zero for decreasePosition

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.MustNewDecFromStr("-99.75"), resp.Position.Size_)       // -105 + 5.25
				require.EqualValues(t, sdk.MustNewDecFromStr("12.85"), resp.Position.Margin)       // 10.5(old) + 0.25(realized PnL) - (-2.1)(funding payment)
				require.EqualValues(t, sdk.MustNewDecFromStr("99.75"), resp.Position.OpenNotional) // 100(position notional) - 5(notional sold) + 4.75(unrealized PnL)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},

		{
			name: "decrease short position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1 BTC = 1.05 NUSD
			// user has position notional value of 105 NUSD and unrealized PnL of -5 NUSD
			// user decreases position by notional value of 5.25 NUSD
			// user ends up with realized PnL of -0.25 NUSD, unrealized PnL of -4.75 NUSD
			//   position notional value of 99.75 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					Address:                             sample.AccAddress().String(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-100), // -100 BTC
					Margin:                              sdk.NewDec(10),   // 10 NUSD
					OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair("BTC:NUSD"),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(105), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.TokenPair("BTC:NUSD"),
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.MustNewDecFromStr("5.25"),
						/*baseAssetLimit=*/ sdk.NewDec(5),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(5), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("decrease position by 5.25 NUSD in notional value")
				resp, err := perpKeeper.decreasePosition(
					ctx,
					currentPosition,
					/*openNotional=*/ sdk.MustNewDecFromStr("5.25"), // NUSD
					/*baseLimit=*/ sdk.NewDec(5), // BTC
					/*canOverFluctuationLimit=*/ false,
				)

				require.NoError(t, err)
				require.EqualValues(t, sdk.MustNewDecFromStr("5.25"), resp.ExchangedQuoteAssetAmount) // open notional
				require.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				require.EqualValues(t, sdk.NewDec(5), resp.ExchangedPositionSize) // sold back to vpool
				require.EqualValues(t, sdk.NewDec(-2), resp.FundingPayment)
				require.EqualValues(t, sdk.ZeroDec(), resp.MarginToVault)
				require.EqualValues(t, sdk.MustNewDecFromStr("-0.25"), resp.RealizedPnl)
				require.EqualValues(t, sdk.MustNewDecFromStr("-4.75"), resp.UnrealizedPnlAfter)

				require.EqualValues(t, currentPosition.Address, resp.Position.Address)
				require.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				require.EqualValues(t, sdk.NewDec(-95), resp.Position.Size_)                 // -100 + 5
				require.EqualValues(t, sdk.MustNewDecFromStr("11.75"), resp.Position.Margin) // 10(old) + (-0.25)(realized PnL) - (-2)(funding payment)
				require.EqualValues(t, sdk.NewDec(95), resp.Position.OpenNotional)           // 105(position notional) - 5.25(notional sold) + (-4.75)(unrealized PnL)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				require.EqualValues(t, 0, resp.Position.LiquidityHistoryIndex)
				require.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		// TODO(https://github.com/NibiruChain/nibiru/issues/361): Add test cases that result in bad debt
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}
