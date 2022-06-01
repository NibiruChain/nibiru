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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

var BtcNusdPair = common.AssetPair{
	Token0: "BTC",
	Token1: "NUSD",
}

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

				tokenPair, err := common.NewAssetPairFromStr(pair)
				require.NoError(t, err)
				latestCumulativePremiumFraction, err := keeper.
					getLatestCumulativePremiumFraction(ctx, tokenPair)
				require.NoError(t, err)

				assert.Equal(t, sdk.NewDec(2), latestCumulativePremiumFraction)
			},
		},
		{
			name: "uninitialized vpool has no metadata | fail",
			test: func() {
				perpKeeper, _, ctx := getKeeper(t)
				vpool := common.AssetPair{
					Token0: "xxx",
					Token1: "yyy",
				}
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

type mockedDependencies struct {
	mockAccountKeeper   *mock.MockAccountKeeper
	mockBankKeeper      *mock.MockBankKeeper
	mockPricefeedKeeper *mock.MockPricefeedKeeper
	mockVpoolKeeper     *mock.MockVpoolKeeper
}

func getKeeper(t *testing.T) (Keeper, mockedDependencies, sdk.Context) {
	db := tmdb.NewMemDB()
	commitMultiStore := store.NewCommitMultiStore(db)
	// Mount the KV store with the x/perp store key
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	commitMultiStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	// Mount Transient store
	transientStoreKey := sdk.NewTransientStoreKey("transient" + types.StoreKey)
	commitMultiStore.MountStoreWithDB(transientStoreKey, sdk.StoreTypeTransient, nil)
	// Mount Memory store
	memStoreKey := storetypes.NewMemoryStoreKey("mem" + types.StoreKey)
	commitMultiStore.MountStoreWithDB(memStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, commitMultiStore.LoadLatestVersion())

	protoCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	params := initParamsKeeper(
		protoCodec, codec.NewLegacyAmino(), storeKey, memStoreKey)

	subSpace, found := params.GetSubspace(types.ModuleName)
	require.True(t, found)

	ctrl := gomock.NewController(t)
	mockedAccountKeeper := mock.NewMockAccountKeeper(ctrl)
	mockedBankKeeper := mock.NewMockBankKeeper(ctrl)
	mockedPricefeedKeeper := mock.NewMockPricefeedKeeper(ctrl)
	mockedVpoolKeeper := mock.NewMockVpoolKeeper(ctrl)

	mockedAccountKeeper.
		EXPECT().GetModuleAddress(types.ModuleName).
		Return(authtypes.NewModuleAddress(types.ModuleName))

	k := NewKeeper(
		protoCodec,
		storeKey,
		subSpace,
		mockedAccountKeeper,
		mockedBankKeeper,
		mockedPricefeedKeeper,
		mockedVpoolKeeper,
	)

	ctx := sdk.NewContext(commitMultiStore, tmproto.Header{}, false, log.NewNopLogger())

	return k, mockedDependencies{
		mockAccountKeeper:   mockedAccountKeeper,
		mockBankKeeper:      mockedBankKeeper,
		mockPricefeedKeeper: mockedPricefeedKeeper,
		mockVpoolKeeper:     mockedVpoolKeeper,
	}, ctx
}

func initParamsKeeper(
	appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino,
	key sdk.StoreKey, tkey sdk.StoreKey,
) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)
	paramsKeeper.Subspace(types.ModuleName)

	return paramsKeeper
}

func TestGetPositionNotionalAndUnrealizedPnl(t *testing.T) {
	tests := []struct {
		name                       string
		initialPosition            types.Position
		setMocks                   func(ctx sdk.Context, mocks mockedDependencies)
		pnlCalcOption              types.PnLCalcOption
		expectedPositionalNotional sdk.Dec
		expectedUnrealizedPnL      sdk.Dec
	}{
		{
			name: "long position; positive pnl; spot price calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(10),
		},
		{
			name: "long position; negative pnl; spot price calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(5), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(-5),
		},
		{
			name: "long position; positive pnl; twap calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(20), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_TWAP,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(10),
		},
		{
			name: "long position; negative pnl; twap calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(5), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_TWAP,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(-5),
		},
		{
			name: "long position; positive pnl; oracle calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetUnderlyingPrice(
						ctx,
						BtcNusdPair,
					).
					Return(sdk.NewDec(2), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_ORACLE,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(10),
		},
		{
			name: "long position; negative pnl; oracle calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetUnderlyingPrice(
						ctx,
						BtcNusdPair,
					).
					Return(sdk.MustNewDecFromStr("0.5"), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_ORACLE,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(-5),
		},
		{
			name: "short position; positive pnl; spot price calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(5), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(5),
		},
		{
			name: "short position; negative pnl; spot price calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(-10),
		},
		{
			name: "short position; positive pnl; twap calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(5), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_TWAP,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(5),
		},
		{
			name: "short position; negative pnl; twap calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(20), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_TWAP,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(-10),
		},
		{
			name: "short position; positive pnl; oracle calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetUnderlyingPrice(
						ctx,
						BtcNusdPair,
					).
					Return(sdk.MustNewDecFromStr("0.5"), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_ORACLE,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(5),
		},
		{
			name: "long position; negative pnl; oracle calc",
			initialPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetUnderlyingPrice(
						ctx,
						BtcNusdPair,
					).
					Return(sdk.NewDec(2), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_ORACLE,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(-10),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			tc.setMocks(ctx, mocks)

			positionalNotional, unrealizedPnl, err := perpKeeper.
				getPositionNotionalAndUnrealizedPnL(
					ctx,
					tc.initialPosition,
					tc.pnlCalcOption,
				)
			require.NoError(t, err)

			assert.EqualValues(t, tc.expectedPositionalNotional, positionalNotional)
			assert.EqualValues(t, tc.expectedUnrealizedPnL, unrealizedPnl)
		})
	}
}

func TestSwapQuoteAssetForBase(t *testing.T) {
	tests := []struct {
		name               string
		setMocks           func(ctx sdk.Context, mocks mockedDependencies)
		side               types.Side
		expectedBaseAmount sdk.Dec
	}{
		{
			name: "long position - buy",
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAmount=*/ sdk.NewDec(10),
						/*baseLimit=*/ sdk.NewDec(1),
					).Return(sdk.NewDec(5), nil)
			},
			side:               types.Side_BUY,
			expectedBaseAmount: sdk.NewDec(5),
		},
		{
			name: "short position - sell",
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAmount=*/ sdk.NewDec(10),
						/*baseLimit=*/ sdk.NewDec(1),
					).Return(sdk.NewDec(5), nil)
			},
			side:               types.Side_SELL,
			expectedBaseAmount: sdk.NewDec(-5),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			tc.setMocks(ctx, mocks)

			baseAmount, err := perpKeeper.swapQuoteForBase(
				ctx,
				BtcNusdPair,
				tc.side,
				sdk.NewDec(10),
				sdk.NewDec(1),
				false,
			)

			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedBaseAmount, baseAmount)
		})
	}
}

func TestGetPreferencePositionNotionalAndUnrealizedPnl(t *testing.T) {
	// all tests are assumed long positions with positive pnl for ease of calculation
	// short positions and negative pnl are implicitly correct because of
	// TestGetPositionNotionalAndUnrealizedPnl
	testcases := []struct {
		name                       string
		initPosition               types.Position
		setMocks                   func(ctx sdk.Context, mocks mockedDependencies)
		pnlPreferenceOption        types.PnLPreferenceOption
		expectedPositionalNotional sdk.Dec
		expectedUnrealizedPnl      sdk.Dec
	}{
		{
			name: "max pnl, pick spot price",
			initPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				t.Log("Mock vpool spot price")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(15), nil)
			},
			pnlPreferenceOption:        types.PnLPreferenceOption_MAX,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnl:      sdk.NewDec(10),
		},
		{
			name: "max pnl, pick twap",
			initPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				t.Log("Mock vpool spot price")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(30), nil)
			},
			pnlPreferenceOption:        types.PnLPreferenceOption_MAX,
			expectedPositionalNotional: sdk.NewDec(30),
			expectedUnrealizedPnl:      sdk.NewDec(20),
		},
		{
			name: "min pnl, pick spot price",
			initPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				t.Log("Mock vpool spot price")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(30), nil)
			},
			pnlPreferenceOption:        types.PnLPreferenceOption_MIN,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnl:      sdk.NewDec(10),
		},
		{
			name: "min pnl, pick twap",
			initPosition: types.Position{
				TraderAddress: sample.AccAddress(),
				Pair:          "BTC:NUSD",
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				t.Log("Mock vpool spot price")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(15), nil)
			},
			pnlPreferenceOption:        types.PnLPreferenceOption_MIN,
			expectedPositionalNotional: sdk.NewDec(15),
			expectedUnrealizedPnl:      sdk.NewDec(5),
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			tc.setMocks(ctx, mocks)

			positionalNotional, unrealizedPnl, err := perpKeeper.
				getPreferencePositionNotionalAndUnrealizedPnL(
					ctx,
					tc.initPosition,
					tc.pnlPreferenceOption,
				)

			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedPositionalNotional, positionalNotional)
			assert.EqualValues(t, tc.expectedUnrealizedPnl, unrealizedPnl)
		})
	}
}

func TestIncreasePosition(t *testing.T) {
	tests := []struct {
		name         string
		initPosition types.Position
		given        func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper)
		when         func(ctx sdk.Context, perpKeeper Keeper, initPosition types.Position) (*types.PositionResp, error)
		then         func(t *testing.T, ctx sdk.Context, initPosition types.Position, resp *types.PositionResp, err error)
	}{
		{
			name: "increase long position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1BTC=1NUSD)
			// BTC went up in value, now its price is 1BTC=2NUSD
			// user increases position by another 10 NUSD at 10x leverage
			initPosition: types.Position{
				TraderAddress:                       sample.AccAddress(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(100), // 100 BTC
				Margin:                              sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                        sdk.NewDec(100), // 100 NUSD
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(50),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(50), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
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
			},
			when: func(ctx sdk.Context, perpKeeper Keeper, initPosition types.Position) (*types.PositionResp, error) {
				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				return perpKeeper.increasePosition(
					ctx,
					initPosition,
					types.Side_BUY,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(50), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)
			},
			then: func(t *testing.T, ctx sdk.Context, initPosition types.Position, resp *types.PositionResp, err error) {
				require.NoError(t, err)
				assert.True(t, sdk.NewDec(100).Equal(resp.ExchangedQuoteAssetAmount))
				assert.True(t, sdk.ZeroDec().Equal(resp.BadDebt))
				assert.EqualValues(t, sdk.NewDec(50), resp.ExchangedPositionSize)
				assert.True(t, sdk.NewDec(2).Equal(resp.FundingPayment))
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)
				assert.True(t, sdk.NewDec(10).Equal(resp.MarginToVault))
				assert.EqualValues(t, sdk.NewDec(100), resp.UnrealizedPnlAfter)

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(150), resp.Position.Size_)        // 100 + 50
				assert.True(t, sdk.NewDec(18).Equal(resp.Position.Margin))         // 10(old) + 10(new) - 2(funding payment)
				assert.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "increase long position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1BTC=1NUSD)
			// BTC went down in value, now its price is 1.01BTC=1NUSD
			// user increases position by another 10 NUSD at 10x leverage
			initPosition: types.Position{
				TraderAddress:                       sample.AccAddress(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(100), // 100 BTC
				Margin:                              sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                        sdk.NewDec(100), // 100 NUSD
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(101),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(101), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
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
			},
			when: func(ctx sdk.Context, perpKeeper Keeper, initPosition types.Position) (*types.PositionResp, error) {
				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				return perpKeeper.increasePosition(
					ctx,
					initPosition,
					types.Side_BUY,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(101), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)
			},
			then: func(t *testing.T, ctx sdk.Context, initPosition types.Position, resp *types.PositionResp, err error) {
				require.NoError(t, err)
				assert.True(t, sdk.NewDec(100).Equal(resp.ExchangedQuoteAssetAmount)) // equal to open notional
				assert.True(t, sdk.ZeroDec().Equal(resp.BadDebt))
				assert.EqualValues(t, sdk.NewDec(101), resp.ExchangedPositionSize) // equal to base amount bought
				assert.True(t, sdk.NewDec(2).Equal(resp.FundingPayment))           // 0.02 * 100
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)             // always zero for increasePosition
				assert.True(t, sdk.NewDec(10).Equal(resp.MarginToVault))           // openNotional / leverage
				assert.EqualValues(t, sdk.NewDec(-1), resp.UnrealizedPnlAfter)     // 99 - 100

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(201), resp.Position.Size_)        // 100 + 101
				assert.True(t, sdk.NewDec(18).Equal(resp.Position.Margin))         // 10(old) + 10(new) - 2(funding payment)
				assert.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
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
			initPosition: types.Position{
				TraderAddress:                       sample.AccAddress(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(110), // 110 BTC
				Margin:                              sdk.NewDec(11),  // 11 NUSD
				OpenNotional:                        sdk.NewDec(110), // 110 NUSD
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(110),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(110), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
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
			},
			when: func(ctx sdk.Context, perpKeeper Keeper, initPosition types.Position) (*types.PositionResp, error) {
				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				return perpKeeper.increasePosition(
					ctx,
					initPosition,
					types.Side_BUY,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(110), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)
			},
			then: func(t *testing.T, ctx sdk.Context, initPosition types.Position, resp *types.PositionResp, err error) {
				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(10), resp.MarginToVault) // openNotional / leverage
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)    // always zero for increasePosition

				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // equal to open notional
				assert.EqualValues(t, sdk.NewDec(110), resp.ExchangedPositionSize)     // equal to base amount bought
				assert.EqualValues(t, sdk.NewDec(22), resp.FundingPayment)             // 0.02 * 110
				assert.EqualValues(t, sdk.NewDec(-10), resp.UnrealizedPnlAfter)        // 90 - 100
				assert.EqualValues(t, sdk.NewDec(1), resp.BadDebt)                     // 11(old) + 10(new) - 22(funding payment)

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(220), resp.Position.Size_)        // 110 + 110
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)         // 11(old) + 10(new) - 22(funding payment) --> zero margin left
				assert.EqualValues(t, sdk.NewDec(210), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.2"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "increase short position, positive PnL",
			// user sold 100 BTC for 100 NUSD at 10x leverage (1BTC=1NUSD)
			// user's initial margin deposit was 10 NUSD
			// BTC went down in value, now its price is 2BTC=1NUSD
			// user increases position by another 10 NUSD at 10x leverage
			initPosition: types.Position{
				TraderAddress:                       sample.AccAddress(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(-100), // -100 BTC
				Margin:                              sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(200),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
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
			},
			when: func(ctx sdk.Context, perpKeeper Keeper, initPosition types.Position) (*types.PositionResp, error) {
				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				return perpKeeper.increasePosition(
					ctx,
					initPosition,
					types.Side_SELL,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(200), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)
			},
			then: func(t *testing.T, ctx sdk.Context, initPosition types.Position, resp *types.PositionResp, err error) {
				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // equal to open notional
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				assert.EqualValues(t, sdk.NewDec(-200), resp.ExchangedPositionSize) // equal to amount of base asset IOUs
				assert.EqualValues(t, sdk.NewDec(-2), resp.FundingPayment)          // -100 * 0.02
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)              // always zero for increasePosition
				assert.EqualValues(t, sdk.NewDec(10), resp.MarginToVault)           // open notional / leverage
				assert.EqualValues(t, sdk.NewDec(50), resp.UnrealizedPnlAfter)      // 100 - 50

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-300), resp.Position.Size_)       // -100 - 200
				assert.EqualValues(t, sdk.NewDec(22), resp.Position.Margin)        // 10(old) + 10(new)  - (-2)(funding payment)
				assert.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "increase short position, negative PnL",
			// user sold 100 BTC for 100 NUSD at 10x leverage (1BTC=1NUSD)
			// user's initial margin deposit was 10 NUSD
			// BTC went up in value, now its price is 0.99BTC=1NUSD
			// user increases position by another 10 NUSD at 10x leverage
			initPosition: types.Position{
				TraderAddress:                       sample.AccAddress(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(-100), // 100 BTC
				Margin:                              sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(99),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(99), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
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
			},
			when: func(ctx sdk.Context, perpKeeper Keeper, initPosition types.Position) (*types.PositionResp, error) {
				t.Log("Increase position with 10 NUSD margin and 10x leverage.")
				return perpKeeper.increasePosition(
					ctx,
					initPosition,
					types.Side_SELL,
					/*openNotional=*/ sdk.NewDec(100), // NUSD
					/*baseLimit=*/ sdk.NewDec(99), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)
			},
			then: func(t *testing.T, ctx sdk.Context, initPosition types.Position, resp *types.PositionResp, err error) {
				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // equal to open notional
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				assert.EqualValues(t, sdk.NewDec(-99), resp.ExchangedPositionSize) // base asset IOUs
				assert.EqualValues(t, sdk.NewDec(-2), resp.FundingPayment)         // -100 * 0.02
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)             // always zero for increasePosition
				assert.EqualValues(t, sdk.NewDec(10), resp.MarginToVault)          // openNotional / leverage
				assert.EqualValues(t, sdk.NewDec(-1), resp.UnrealizedPnlAfter)     // 100 - 101

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-199), resp.Position.Size_)       // -100 - 99
				assert.EqualValues(t, sdk.NewDec(22), resp.Position.Margin)        // 10(old) + 10(new) - (-2)(funding payment)
				assert.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
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
			initPosition: types.Position{
				TraderAddress:                       sample.AccAddress(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(-100), // 100 BTC
				Margin:                              sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(105),
						/*baseAssetLimit=*/ sdk.NewDec(100),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
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
			},
			when: func(ctx sdk.Context, perpKeeper Keeper, initPosition types.Position) (*types.PositionResp, error) {
				t.Log("Increase position with 10.5 NUSD margin and 10x leverage.")
				return perpKeeper.increasePosition(
					ctx,
					initPosition,
					types.Side_SELL,
					/*openNotional=*/ sdk.NewDec(105), // NUSD
					/*baseLimit=*/ sdk.NewDec(100), // BTC
					/*leverage=*/ sdk.NewDec(10),
				)
			},
			then: func(t *testing.T, ctx sdk.Context, initPosition types.Position, resp *types.PositionResp, err error) {
				require.NoError(t, err)
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)                                     // always zero for increasePosition
				assert.EqualValues(t, sdk.MustNewDecFromStr("10.5").String(), resp.MarginToVault.String()) // openNotional / leverage

				assert.EqualValues(t, sdk.NewDec(105), resp.ExchangedQuoteAssetAmount)              // equal to open notional
				assert.EqualValues(t, sdk.NewDec(-100), resp.ExchangedPositionSize)                 // base asset IOUs
				assert.EqualValues(t, sdk.NewDec(30), resp.FundingPayment)                          // -100 * (-0.2)
				assert.EqualValues(t, sdk.NewDec(-5), resp.UnrealizedPnlAfter)                      // 100 - 105
				assert.EqualValues(t, sdk.MustNewDecFromStr("9.5").String(), resp.BadDebt.String()) // 10(old) + 10.5(new) - (30)(funding payment)

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-200), resp.Position.Size_)       // -100 + (-100)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)         // 10(old) + 10.5(new) - (30)(funding payment) --> zero margin left
				assert.EqualValues(t, sdk.NewDec(205), resp.Position.OpenNotional) // 100(old) + 105(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("-0.3"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			tc.given(ctx, mocks, perpKeeper)

			resp, err := tc.when(ctx, perpKeeper, tc.initPosition)

			tc.then(t, ctx, tc.initPosition, resp, err)
		})
	}
}

func TestClosePositionEntirely(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		/*==========================LONG POSITIONS============================*/
		{
			name: "close long position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(10),  // 10 NUSD
					OpenNotional:                        sdk.NewDec(100), // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.NewDec(200),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closePositionEntirely(
					ctx,
					currentPosition,
					/*quoteAssetLimit=*/ sdk.NewDec(200), // NUSD
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(200).String(), resp.ExchangedQuoteAssetAmount.String()) // amount of quote obtained
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				assert.EqualValues(t, sdk.NewDec(-100), resp.ExchangedPositionSize) // sold back to vpool
				assert.EqualValues(t, sdk.NewDec(2), resp.FundingPayment)
				assert.EqualValues(t, sdk.NewDec(-108).String(), resp.MarginToVault.String()) // ( 10(oldMargin) + 100(unrealzedPnL) - 2(fundingPayment) ) * -1
				assert.EqualValues(t, sdk.NewDec(100), resp.RealizedPnl)
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter) // always zero when closing a position entirely

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Size_)        // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)       // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.OpenNotional) // always zero
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "close long position, negative PnL",
			// user bought in at 100 BTC for 10.5 NUSD at 10x leverage (1 BTC = 1.05 NUSD)
			// notional value is 105 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -5 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(12),  // 10.5 NUSD
					OpenNotional:                        sdk.NewDec(120), // 105 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.NewDec(100),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closePositionEntirely(
					ctx,
					currentPosition,
					/*quoteAssetLimit=*/ sdk.NewDec(100), // NUSD
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(100).String(), resp.ExchangedQuoteAssetAmount.String()) // amount of quote obtained
				assert.EqualValues(t, sdk.NewDec(10).String(), resp.BadDebt.String())
				assert.EqualValues(t, sdk.NewDec(-100), resp.ExchangedPositionSize) // sold back to vpool
				assert.EqualValues(t, sdk.NewDec(2).String(), resp.FundingPayment.String())
				assert.EqualValues(t, sdk.NewDec(0).String(), resp.MarginToVault.String()) // ( 10.5(oldMargin) + (-5)(unrealzedPnL) - 2(fundingPayment) ) * -1
				assert.EqualValues(t, sdk.NewDec(-20), resp.RealizedPnl)
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter) // always zero when closing a position entirely

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Size_)        // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)       // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.OpenNotional) // always zero
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "close long position, negative PnL leads to bad debt",
			// user bought in at 100 BTC for 15 NUSD at 10x leverage (1 BTC = 1.5 NUSD)
			// notional value is 150 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -50 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(15),  // 15 NUSD
					OpenNotional:                        sdk.NewDec(150), // 150 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.NewDec(100),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closePositionEntirely(
					ctx,
					currentPosition,
					/*quoteAssetLimit=*/ sdk.NewDec(100), // NUSD
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // amount of quote obtained
				assert.EqualValues(t, sdk.NewDec(-100), resp.ExchangedPositionSize)    // sold back to vpool
				assert.EqualValues(t, sdk.NewDec(2), resp.FundingPayment)
				assert.EqualValues(t, sdk.ZeroDec(), resp.MarginToVault) // ( 15(oldMargin) + (-50)(unrealzedPnL) - 2(fundingPayment) ) * -1, clippsed at zero
				assert.EqualValues(t, sdk.NewDec(-50), resp.RealizedPnl)
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter) // always zero when closing a position entirely
				assert.EqualValues(t, sdk.NewDec(37), resp.BadDebt)           // 15(oldMargin) + (-50)(unrealzedPnL) - 2(fundingPayment)

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Size_)        // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)       // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.OpenNotional) // always zero
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},

		/*==========================SHORT POSITIONS===========================*/
		{
			name: "close short position, positive PnL",
			// user bought in at 150 BTC for 15 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 150 NUSD
			// BTC drops in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of +50 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-150), // -150 BTC
					Margin:                              sdk.NewDec(15),   // 15 NUSD
					OpenNotional:                        sdk.NewDec(150),  // 150 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(150),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*baseAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(150),
						/*quoteAssetLimit=*/ sdk.NewDec(100),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closePositionEntirely(
					ctx,
					currentPosition,
					/*quoteAssetLimit=*/ sdk.NewDec(100), // NUSD
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // amount of quote used to purchase
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				assert.EqualValues(t, sdk.NewDec(150), resp.ExchangedPositionSize) // bought back from vpool
				assert.EqualValues(t, sdk.NewDec(-3), resp.FundingPayment)         // -150 * 0.02
				assert.EqualValues(t, sdk.NewDec(50), resp.RealizedPnl)            // 150 - 100
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter)
				assert.EqualValues(t, sdk.NewDec(-68), resp.MarginToVault) // ( 15(oldMargin) + 50(PnL) - (-3)(fundingPayment) ) * -1

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Size_)        // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)       // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.OpenNotional) // always zero
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "close short position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1.05 BTC = 1 NUSD
			// user has position notional value of 105 NUSD and unrealized PnL of -5 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-100), // -100 BTC
					Margin:                              sdk.NewDec(10),   // 10 NUSD
					OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(105), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*baseAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.NewDec(105),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(105), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closePositionEntirely(
					ctx,
					currentPosition,
					/*quoteAssetLimit=*/ sdk.NewDec(105), // NUSD
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(105), resp.ExchangedQuoteAssetAmount) // amount of quote used to purchase
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedPositionSize)     // bought back from vpool
				assert.EqualValues(t, sdk.NewDec(-2), resp.FundingPayment)             // -100 * 0.02
				assert.EqualValues(t, sdk.NewDec(-5), resp.RealizedPnl)                // 100 - 105
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter)          // always zero
				assert.EqualValues(t, sdk.NewDec(-7), resp.MarginToVault)              // ( 10(oldMargin) + (-5)(PnL) - (-2)(fundingPayment) ) * -1
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Size_)        // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)       // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.OpenNotional) // always zero
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "close short position, negative PnL leads to bad debt",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 150 NUSD and unrealized PnL of -50 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-100), // -100 BTC
					Margin:                              sdk.NewDec(10),   // 10 NUSD
					OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(150), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*baseAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.NewDec(150),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(150), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closePositionEntirely(
					ctx,
					currentPosition,
					/*quoteAssetLimit=*/ sdk.NewDec(150), // NUSD
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(150), resp.ExchangedQuoteAssetAmount) // amount of quote used to purchase
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedPositionSize)     // bought back from vpool
				assert.True(t, sdk.NewDec(-2).Equal(resp.FundingPayment))              // -100 * 0.02
				assert.EqualValues(t, sdk.NewDec(-50), resp.RealizedPnl)               // 100 - 105
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter)          // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.MarginToVault)               // ( 10(oldMargin) + (-50)(PnL) - (-2)(fundingPayment) ) * -1 --> clipped to zero
				assert.EqualValues(t, sdk.NewDec(38), resp.BadDebt)                    // 10(oldMargin) + (-50)(PnL) - (-2)(fundingPayment)

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Size_)        // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)       // always zero
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.OpenNotional) // always zero
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
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
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(10),  // 10 NUSD
					OpenNotional:                        sdk.NewDec(100), // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
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
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedQuoteAssetAmount) // open notional
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				assert.EqualValues(t, sdk.NewDec(-50), resp.ExchangedPositionSize) // sold back to vpool
				assert.EqualValues(t, sdk.NewDec(2), resp.FundingPayment)
				assert.EqualValues(t, sdk.ZeroDec(), resp.MarginToVault)
				assert.EqualValues(t, sdk.NewDec(50), resp.RealizedPnl)
				assert.EqualValues(t, sdk.NewDec(50), resp.UnrealizedPnlAfter)

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(50), resp.Position.Size_)        // 100 - 50
				assert.EqualValues(t, sdk.NewDec(58), resp.Position.Margin)       // 10(old) + 50(realized PnL) - 2(funding payment)
				assert.EqualValues(t, sdk.NewDec(50), resp.Position.OpenNotional) // 200(position notional) - 100(notional sold) - 50(unrealized PnL)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
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
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(105), // 105 BTC
					Margin:                              sdk.NewDec(11),  // 10.5 NUSD
					OpenNotional:                        sdk.NewDec(110), // 105 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(105),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
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
				assert.EqualValues(t, sdk.NewDec(5), resp.ExchangedQuoteAssetAmount) // open notional
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				assert.EqualValues(t, sdk.MustNewDecFromStr("-5.25"), resp.ExchangedPositionSize) // sold back to vpool
				// resp.FundingPayment = pos.Size_ * fundingDifference
				//   = 105 * 0.02 - 2.1 → 2
				assert.EqualValues(t, sdk.MustNewDecFromStr("2.1").String(), resp.FundingPayment.String())
				assert.EqualValues(t, sdk.MustNewDecFromStr("-0.5").String(), resp.RealizedPnl.String())        // (-5)(unrealizedPnL) * 5.25/105 (fraction of position size reduced)
				assert.EqualValues(t, sdk.MustNewDecFromStr("-9.5").String(), resp.UnrealizedPnlAfter.String()) // (-5)(unrealizedPnL) - (-0.25)(realizedPnL)
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.MarginToVault.String())                      // always zero for decreasePosition

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.MustNewDecFromStr("99.75"), resp.Position.Size_)                  // 105 - 5.25
				assert.EqualValues(t, sdk.MustNewDecFromStr("8.4").String(), resp.Position.Margin.String()) // 10(old) + (-0.25)(realized PnL) - 2.1(funding payment)
				assert.EqualValues(t,
					sdk.MustNewDecFromStr("104.5").String(),
					resp.Position.OpenNotional.String()) // 100(position notional) - 5(notional sold) - (-4.75)(unrealized PnL)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
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
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-105), // -105 BTC
					Margin:                              sdk.NewDec(11),   // 10.5 NUSD
					OpenNotional:                        sdk.NewDec(110),  // 105 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(105),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
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
				assert.EqualValues(t, sdk.NewDec(5), resp.ExchangedQuoteAssetAmount) // open notional
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				assert.EqualValues(t, sdk.MustNewDecFromStr("5.25"), resp.ExchangedPositionSize)            // bought back from vpool
				assert.EqualValues(t, sdk.MustNewDecFromStr("-2.1").String(), resp.FundingPayment.String()) // -105 * 0.02
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.5"), resp.RealizedPnl)                       // (-5)(unrealizedPnL) * 5.25/105 (fraction of position size reduced)
				assert.EqualValues(t, sdk.MustNewDecFromStr("9.5"), resp.UnrealizedPnlAfter)                // (-5)(unrealizedPnL) - (-0.25)(realizedPnL)
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.MarginToVault.String())                  // always zero for decreasePosition

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.MustNewDecFromStr("-99.75"), resp.Position.Size_)                         // -105 + 5.25
				assert.EqualValues(t, sdk.MustNewDecFromStr("13.6").String(), resp.Position.Margin.String())        // 10.5(old) + 0.25(realized PnL) - (-2.1)(funding payment)
				assert.EqualValues(t, sdk.MustNewDecFromStr("104.5").String(), resp.Position.OpenNotional.String()) // 100(position notional) - 5(notional sold) + 4.75(unrealized PnL)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
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
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-100), // -100 BTC
					Margin:                              sdk.NewDec(10),   // 10 NUSD
					OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(105), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
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
				assert.EqualValues(t, sdk.MustNewDecFromStr("5.25").String(), resp.ExchangedQuoteAssetAmount.String()) // open notional
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.BadDebt.String())
				assert.EqualValues(t, sdk.NewDec(5), resp.ExchangedPositionSize) // sold back to vpool
				assert.EqualValues(t, sdk.NewDec(-2).String(), resp.FundingPayment.String())
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.MarginToVault.String())
				assert.EqualValues(t, sdk.MustNewDecFromStr("-0.25"), resp.RealizedPnl)
				assert.EqualValues(t, sdk.MustNewDecFromStr("-4.75"), resp.UnrealizedPnlAfter)

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-95), resp.Position.Size_)                                   // -100 + 5
				assert.EqualValues(t, sdk.MustNewDecFromStr("11.75").String(), resp.Position.Margin.String()) // 10(old) + (-0.25)(realized PnL) - (-2)(funding payment)
				assert.EqualValues(t, sdk.NewDec(95), resp.Position.OpenNotional)                             // 105(position notional) - 5.25(notional sold) + (-4.75)(unrealized PnL)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
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

func TestCloseAndOpenReversePosition(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		/*==========================LONG POSITIONS============================*/
		{
			name: "close long position, positive PnL, open short position",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user closes position and opens in reverse direction with 30*10 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(10),  // 10 NUSD
					OpenNotional:                        sdk.NewDec(100), // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(50),
					).Return( /*baseAssetLimit=*/ sdk.NewDec(50), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position and open reverse")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(30), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetLimit=*/ sdk.NewDec(150),
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(300).String(), resp.ExchangedQuoteAssetAmount.String()) // 30 * 10
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.BadDebt.String())
				assert.EqualValues(t, sdk.NewDec(-150), resp.ExchangedPositionSize)          // 100 original + 50 shorted
				assert.EqualValues(t, sdk.NewDec(2).String(), resp.FundingPayment.String())  // 100 * 0.02
				assert.EqualValues(t, sdk.NewDec(-98).String(), resp.MarginToVault.String()) // -1 * ( 10(oldMargin) + 100(unrealzedPnL) - 2(fundingPayment) ) + 10
				assert.EqualValues(t, sdk.NewDec(100), resp.RealizedPnl)
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter) // always zero

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-50), resp.Position.Size_)
				assert.EqualValues(t, sdk.NewDec(10).String(), resp.Position.Margin.String())
				assert.EqualValues(t, sdk.NewDec(100), resp.Position.OpenNotional)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "close long position, negative PnL, open short position",
			// user bought in at 100 BTC for 10.5 NUSD at 10x leverage (1 BTC = 1.05 NUSD)
			// notional value is 105 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -5 NUSD
			// user closes position and opens in reverse direction with 30*10 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(11),  // 10.5 NUSD
					OpenNotional:                        sdk.NewDec(105), // 105 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(100),
					).Return( /*baseAssetLimit=*/ sdk.NewDec(100), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(20), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetLimit=*/ sdk.NewDec(200),
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(200).String(), resp.ExchangedQuoteAssetAmount.String()) // 20 * 10
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.BadDebt.String())
				assert.EqualValues(t, sdk.NewDec(-200), resp.ExchangedPositionSize)         // 100 original + 50 shorted
				assert.EqualValues(t, sdk.NewDec(2).String(), resp.FundingPayment.String()) // 100 * 0.02
				// resp.MarginToVault
				// = -1 * (oldMargin + unrealizedPnL - fundingPayment) + 10
				// = -1 * (11  - 5 - 2 ) + 10
				// =  				  -4 + 10  = 6
				assert.EqualValues(t, sdk.NewDec(6).String(), resp.MarginToVault.String())
				assert.EqualValues(t, sdk.NewDec(-5), resp.RealizedPnl)
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter) // always zero

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-100), resp.Position.Size_)
				assert.EqualValues(t, sdk.NewDec(10).String(), resp.Position.Margin.String())
				assert.EqualValues(t, sdk.NewDec(100), resp.Position.OpenNotional)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "close long position, negative PnL leads to bad debt, cannot close and open reverse",
			// user bought in at 100 BTC for 15 NUSD at 10x leverage (1 BTC = 1.5 NUSD)
			// notional value is 150 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -50 NUSD
			// user tries to close and open reverse position but cannot because it leads to bad debt
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(15),  // 15 NUSD
					OpenNotional:                        sdk.NewDec(150), // 150 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(20), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetLimit=*/ sdk.NewDec(200),
				)

				require.Error(t, err)
				require.Nil(t, resp)
			},
		},
		{
			name: "existing long position, positive PnL, zero base asset limit",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user closes position and opens in reverse direction with 30*10 NUSD
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(10),  // 10 NUSD
					OpenNotional:                        sdk.NewDec(100), // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*baseAssetLimit=*/ sdk.NewDec(50), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position and open reverse")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(30), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetLimit=*/ sdk.ZeroDec(),
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(300).String(), resp.ExchangedQuoteAssetAmount.String()) // 30 * 10
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.BadDebt.String())
				assert.EqualValues(t, sdk.NewDec(-150), resp.ExchangedPositionSize)          // 100 original + 50 shorted
				assert.EqualValues(t, sdk.NewDec(2).String(), resp.FundingPayment.String())  // 100 * 0.02
				assert.EqualValues(t, sdk.NewDec(-98).String(), resp.MarginToVault.String()) // -1 * ( 10(oldMargin) + 100(unrealzedPnL) - 2(fundingPayment) ) + 10
				assert.EqualValues(t, sdk.NewDec(100), resp.RealizedPnl)
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter) // always zero

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-50), resp.Position.Size_)
				assert.EqualValues(t, sdk.NewDec(10).String(), resp.Position.Margin.String())
				assert.EqualValues(t, sdk.NewDec(100), resp.Position.OpenNotional)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "existing long position, positive PnL, small base asset limit",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user closes position and opens in reverse direction with 30*10 NUSD
			// user is unable to do so since the base asset limit is too small
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(100), // 100 BTC
					Margin:                              sdk.NewDec(10),  // 10 NUSD
					OpenNotional:                        sdk.NewDec(100), // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position and open reverse")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(30), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetLimit=*/ sdk.NewDec(5),
				)

				require.Error(t, err)
				require.Nil(t, resp)
			},
		},

		/*==========================SHORT POSITIONS===========================*/
		{
			name: "close short position, positive PnL",
			// user opened position at 150 BTC for 15 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 150 NUSD
			// BTC drops in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of +50 NUSD
			// user closes and opens position in reverse with 20*10 notional value
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-150), // -150 BTC
					Margin:                              sdk.NewDec(15),   // 15 NUSD
					OpenNotional:                        sdk.NewDec(150),  // 150 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(150),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*baseAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(150),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(150),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(150), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position and open reverse")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(20), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetAmountLimit=*/ sdk.NewDec(300),
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(200).String(), resp.ExchangedQuoteAssetAmount.String()) // 20 * 10
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.BadDebt.String())
				assert.EqualValues(t, sdk.NewDec(300), resp.ExchangedPositionSize)           // 150 + 150
				assert.EqualValues(t, sdk.NewDec(-3).String(), resp.FundingPayment.String()) // -150 * 0.02
				assert.EqualValues(t, sdk.NewDec(50), resp.RealizedPnl)                      // 150 - 100
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter)
				assert.EqualValues(t, sdk.NewDec(-58).String(), resp.MarginToVault.String()) // -1 * ( 15(oldMargin) + 50(PnL) - (-3)(fundingPayment) ) + 10

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(150), resp.Position.Size_)
				assert.EqualValues(t, sdk.NewDec(10).String(), resp.Position.Margin.String())
				assert.EqualValues(t, sdk.NewDec(100), resp.Position.OpenNotional)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "close short position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1.05 BTC = 1 NUSD
			// user has position notional value of 105 NUSD and unrealized PnL of -5 NUSD
			// user closes and opens reverse with 21 * 10 notional value
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-100), // -100 BTC
					Margin:                              sdk.NewDec(10),   // 10 NUSD
					OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(105), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*baseAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(105), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(105),
						/*baseAssetLimit=*/ sdk.NewDec(100),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(21), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetAmountLimit=*/ sdk.NewDec(200),
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(210).String(), resp.ExchangedQuoteAssetAmount.String()) // 21 * 10
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.BadDebt.String())
				assert.EqualValues(t, sdk.NewDec(200), resp.ExchangedPositionSize) // 150 + 150
				assert.EqualValues(t, sdk.NewDec(-2), resp.FundingPayment)         // -100 * 0.03
				assert.EqualValues(t, sdk.NewDec(-5), resp.RealizedPnl)            // 150 - 100
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter)
				assert.EqualValues(t, sdk.MustNewDecFromStr("3.5").String(), resp.MarginToVault.String()) // -1 * ( 10(oldMargin) + (-5))(PnL) - (-2)(fundingPayment) ) + 10.5

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(100), resp.Position.Size_)
				assert.EqualValues(t, sdk.MustNewDecFromStr("10.5").String(), resp.Position.Margin.String())
				assert.EqualValues(t, sdk.NewDec(105), resp.Position.OpenNotional)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "close short position, negative PnL leads to bad debt",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 150 NUSD and unrealized PnL of -50 NUSD
			// user tries to close and open reverse position but cannot due to being underwater
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-100), // -100 BTC
					Margin:                              sdk.NewDec(10),   // 10 NUSD
					OpenNotional:                        sdk.NewDec(100),  // 100 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(150), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*baseAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(150), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(21), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetAmountLimit=*/ sdk.NewDec(200),
				)

				require.Error(t, err)
				require.Nil(t, resp)
			},
		},
		{
			name: "close short position, positive PnL, no base amount limit",
			// user opened position at 150 BTC for 15 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 150 NUSD
			// BTC drops in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of +50 NUSD
			// user closes and opens position in reverse with 20*10 notional value
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-150), // -150 BTC
					Margin:                              sdk.NewDec(15),   // 15 NUSD
					OpenNotional:                        sdk.NewDec(150),  // 150 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(150),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*baseAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(150),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						BtcNusdPair,
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*baseAssetAmount=*/ sdk.NewDec(150), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position and open reverse")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(20), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetAmountLimit=*/ sdk.ZeroDec(),
				)

				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(200).String(), resp.ExchangedQuoteAssetAmount.String()) // 20 * 10
				assert.EqualValues(t, sdk.ZeroDec().String(), resp.BadDebt.String())
				assert.EqualValues(t, sdk.NewDec(300), resp.ExchangedPositionSize)           // 150 + 150
				assert.EqualValues(t, sdk.NewDec(-3).String(), resp.FundingPayment.String()) // -150 * 0.02
				assert.EqualValues(t, sdk.NewDec(50), resp.RealizedPnl)                      // 150 - 100
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter)
				assert.EqualValues(t, sdk.NewDec(-58).String(), resp.MarginToVault.String()) // -1 * ( 15(oldMargin) + 50(PnL) - (-3)(fundingPayment) ) + 10

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(150), resp.Position.Size_)
				assert.EqualValues(t, sdk.NewDec(10).String(), resp.Position.Margin.String())
				assert.EqualValues(t, sdk.NewDec(100), resp.Position.OpenNotional)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "close short position, positive PnL, small base asset limit",
			// user opened position at 150 BTC for 15 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 150 NUSD
			// BTC drops in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of +50 NUSD
			// user closes and opens position in reverse with 20*10 notional value
			// user is unable to do so since the base asset limit is too small
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("set up initial position")
				currentPosition := types.Position{
					TraderAddress:                       sample.AccAddress(),
					Pair:                                "BTC:NUSD",
					Size_:                               sdk.NewDec(-150), // -150 BTC
					Margin:                              sdk.NewDec(15),   // 15 NUSD
					OpenNotional:                        sdk.NewDec(150),  // 150 NUSD
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         0,
				}
				perpKeeper.SetPosition(
					ctx,
					currentPosition.GetAssetPair(),
					currentPosition.TraderAddress,
					&currentPosition,
				)

				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						BtcNusdPair,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(150),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						BtcNusdPair,
						/*baseAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(150),
						/*quoteAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("set up pair metadata and last cumulative premium fraction")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: "BTC:NUSD",
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.02"), // 0.02 NUSD / BTC
					},
				})

				t.Log("close position and open reverse")
				resp, err := perpKeeper.closeAndOpenReversePosition(
					ctx,
					currentPosition,
					/*quoteAssetAmount=*/ sdk.NewDec(20), // NUSD
					/*leverage=*/ sdk.NewDec(10),
					/*baseAssetAmountLimit=*/ sdk.NewDec(5),
				)

				require.Error(t, err)
				require.Nil(t, resp)
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
