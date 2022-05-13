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
		// TODO(https://github.com/NibiruChain/nibiru/issues/360): Add test cases that result in bad debt
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}
