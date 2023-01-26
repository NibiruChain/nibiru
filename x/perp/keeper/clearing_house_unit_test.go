package keeper

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	testutilevents "github.com/NibiruChain/nibiru/x/testutil"

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
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type mockedDependencies struct {
	mockAccountKeeper *mock.MockAccountKeeper
	mockBankKeeper    *mock.MockBankKeeper
	mockOracleKeeper  *mock.MockOracleKeeper
	mockVpoolKeeper   *mock.MockVpoolKeeper
	mockEpochKeeper   *mock.MockEpochKeeper
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
	mockedOracleKeeper := mock.NewMockOracleKeeper(ctrl)
	mockedVpoolKeeper := mock.NewMockVpoolKeeper(ctrl)
	mockedEpochKeeper := mock.NewMockEpochKeeper(ctrl)

	mockedAccountKeeper.
		EXPECT().GetModuleAddress(types.ModuleName).
		Return(authtypes.NewModuleAddress(types.ModuleName))

	k := NewKeeper(
		protoCodec,
		storeKey,
		subSpace,
		mockedAccountKeeper,
		mockedBankKeeper,
		mockedOracleKeeper,
		mockedVpoolKeeper,
		mockedEpochKeeper,
	)

	ctx := sdk.NewContext(commitMultiStore, tmproto.Header{}, false, log.NewNopLogger())

	k.SetParams(ctx, types.DefaultParams())

	return k, mockedDependencies{
		mockAccountKeeper: mockedAccountKeeper,
		mockBankKeeper:    mockedBankKeeper,
		mockOracleKeeper:  mockedOracleKeeper,
		mockVpoolKeeper:   mockedVpoolKeeper,
		mockEpochKeeper:   mockedEpochKeeper,
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
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAmount=*/ sdk.NewDec(10),
						/*baseLimit=*/ sdk.NewDec(1),
						/* skipFluctuationLimitCheck */ false,
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
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAmount=*/ sdk.NewDec(10),
						/*baseLimit=*/ sdk.NewDec(1),
						/* skipFluctuationLimitCheck */ false,
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
				common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
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
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                    sdk.NewDec(100), // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(50),
						/* skipFluctuationLimitCheck */ false,
					).Return( /*baseAssetAmount=*/ sdk.NewDec(50), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(200), nil)

				t.Log("set up pair metadata and last cumulative funding rate")
				setPairMetadata(perpKeeper, ctx,
					types.PairMetadata{
						Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
					},
				)
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
				assert.True(t, sdk.NewDec(100).Equal(resp.ExchangedNotionalValue))
				assert.True(t, sdk.ZeroDec().Equal(resp.BadDebt))
				assert.EqualValues(t, sdk.NewDec(50), resp.ExchangedPositionSize)
				assert.True(t, sdk.NewDec(2).Equal(resp.FundingPayment))
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)
				assert.True(t, sdk.NewDec(10).Equal(resp.MarginToVault))
				assert.EqualValues(t, sdk.NewDec(100), resp.UnrealizedPnlAfter)
				assert.EqualValues(t, sdk.NewDec(300), resp.PositionNotional)

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(150), resp.Position.Size_)        // 100 + 50
				assert.True(t, sdk.NewDec(18).Equal(resp.Position.Margin))         // 10(old) + 10(new) - 2(funding payment)
				assert.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LatestCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			},
		},
		{
			name: "increase long position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1BTC=1NUSD)
			// BTC went down in value, now its price is 1.01BTC=1NUSD
			// user increases position by another 10 NUSD at 10x leverage
			initPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                    sdk.NewDec(100), // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(101),
						/* skipFluctuationLimitCheck */ false,
					).Return( /*baseAssetAmount=*/ sdk.NewDec(101), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(99), nil)

				t.Log("set up pair metadata and last cumulative funding rate")
				setPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
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
				assert.True(t, sdk.NewDec(100).Equal(resp.ExchangedNotionalValue)) // equal to open notional
				assert.True(t, sdk.ZeroDec().Equal(resp.BadDebt))
				assert.EqualValues(t, sdk.NewDec(101), resp.ExchangedPositionSize) // equal to base amount bought
				assert.True(t, sdk.NewDec(2).Equal(resp.FundingPayment))           // 0.02 * 100
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)             // always zero for increasePosition
				assert.True(t, sdk.NewDec(10).Equal(resp.MarginToVault))           // openNotional / leverage
				assert.EqualValues(t, sdk.NewDec(-1), resp.UnrealizedPnlAfter)     // 99 - 100
				assert.EqualValues(t, sdk.NewDec(199), resp.PositionNotional)

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(201), resp.Position.Size_)        // 100 + 101
				assert.True(t, sdk.NewDec(18).Equal(resp.Position.Margin))         // 10(old) + 10(new) - 2(funding payment)
				assert.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LatestCumulativePremiumFraction)
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
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(110), // 110 BTC
				Margin:                          sdk.NewDec(11),  // 11 NUSD
				OpenNotional:                    sdk.NewDec(110), // 110 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						/*quoteAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(110),
						/* skipFluctuationLimitCheck */ false,
					).Return( /*baseAssetAmount=*/ sdk.NewDec(110), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(110),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("set up pair metadata and last cumulative funding rate")
				setPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.2"),
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
				assert.EqualValues(t, sdk.NewDec(10), resp.MarginToVault)           // openNotional / leverage
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)              // always zero for increasePosition
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedNotionalValue) // equal to open notional
				assert.EqualValues(t, sdk.NewDec(110), resp.ExchangedPositionSize)  // equal to base amount bought
				assert.EqualValues(t, sdk.NewDec(22), resp.FundingPayment)          // 0.02 * 110
				assert.EqualValues(t, sdk.NewDec(-10), resp.UnrealizedPnlAfter)     // 90 - 100
				assert.EqualValues(t, sdk.NewDec(1), resp.BadDebt)                  // 11(old) + 10(new) - 22(funding payment)
				assert.EqualValues(t, sdk.NewDec(200), resp.PositionNotional)

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(220), resp.Position.Size_)        // 110 + 110
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)         // 11(old) + 10(new) - 22(funding payment) --> zero margin left
				assert.EqualValues(t, sdk.NewDec(210), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.2"), resp.Position.LatestCumulativePremiumFraction)
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
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(200),
						/* skipFluctuationLimitCheck */ false,
					).Return( /*baseAssetAmount=*/ sdk.NewDec(200), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(50), nil)

				t.Log("set up pair metadata and last cumulative funding rate")
				setPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
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
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedNotionalValue) // equal to open notional
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				assert.EqualValues(t, sdk.NewDec(-200), resp.ExchangedPositionSize) // equal to amount of base asset IOUs
				assert.EqualValues(t, sdk.NewDec(-2), resp.FundingPayment)          // -100 * 0.02
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)              // always zero for increasePosition
				assert.EqualValues(t, sdk.NewDec(10), resp.MarginToVault)           // open notional / leverage
				assert.EqualValues(t, sdk.NewDec(50), resp.UnrealizedPnlAfter)      // 100 - 50
				assert.EqualValues(t, sdk.NewDec(150), resp.PositionNotional)

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-300), resp.Position.Size_)       // -100 - 200
				assert.EqualValues(t, sdk.NewDec(22), resp.Position.Margin)        // 10(old) + 10(new)  - (-2)(funding payment)
				assert.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LatestCumulativePremiumFraction)
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
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // 100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(100),
						/*baseAssetLimit=*/ sdk.NewDec(99),
						/* skipFluctuationLimitCheck */ false,
					).Return( /*baseAssetAmount=*/ sdk.NewDec(99), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(101), nil)

				t.Log("set up pair metadata and last cumulative funding rate")
				setPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
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
				assert.EqualValues(t, sdk.NewDec(100), resp.ExchangedNotionalValue) // equal to open notional
				assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
				assert.EqualValues(t, sdk.NewDec(-99), resp.ExchangedPositionSize) // base asset IOUs
				assert.EqualValues(t, sdk.NewDec(-2), resp.FundingPayment)         // -100 * 0.02
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)             // always zero for increasePosition
				assert.EqualValues(t, sdk.NewDec(10), resp.MarginToVault)          // openNotional / leverage
				assert.EqualValues(t, sdk.NewDec(-1), resp.UnrealizedPnlAfter)     // 100 - 101
				assert.EqualValues(t, sdk.NewDec(201), resp.PositionNotional)

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-199), resp.Position.Size_)       // -100 - 99
				assert.EqualValues(t, sdk.NewDec(22), resp.Position.Margin)        // 10(old) + 10(new) - (-2)(funding payment)
				assert.EqualValues(t, sdk.NewDec(200), resp.Position.OpenNotional) // 100(old) + 100(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LatestCumulativePremiumFraction)
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
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // 100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			given: func(ctx sdk.Context, mocks mockedDependencies, perpKeeper Keeper) {
				t.Log("mock vpool")
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						/*quoteAssetDirection=*/ vpooltypes.Direction_REMOVE_FROM_POOL,
						/*quoteAssetAmount=*/ sdk.NewDec(105),
						/*baseAssetLimit=*/ sdk.NewDec(100),
						/* skipFluctuationLimitCheck */ false,
					).Return( /*baseAssetAmount=*/ sdk.NewDec(100), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(105), nil)

				t.Log("set up pair metadata and last cumulative funding rate")
				setPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("-0.3"),
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
				assert.EqualValues(t, sdk.NewDec(105), resp.ExchangedNotionalValue)                        // equal to open notional
				assert.EqualValues(t, sdk.NewDec(-100), resp.ExchangedPositionSize)                        // base asset IOUs
				assert.EqualValues(t, sdk.NewDec(30), resp.FundingPayment)                                 // -100 * (-0.2)
				assert.EqualValues(t, sdk.NewDec(-5), resp.UnrealizedPnlAfter)                             // 100 - 105
				assert.EqualValues(t, sdk.MustNewDecFromStr("9.5").String(), resp.BadDebt.String())        // 10(old) + 10.5(new) - (30)(funding payment)
				assert.EqualValues(t, sdk.NewDec(210), resp.PositionNotional)

				assert.EqualValues(t, initPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, initPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.NewDec(-200), resp.Position.Size_)       // -100 + (-100)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)         // 10(old) + 10.5(new) - (30)(funding payment) --> zero margin left
				assert.EqualValues(t, sdk.NewDec(205), resp.Position.OpenNotional) // 100(old) + 105(new)
				assert.EqualValues(t, sdk.MustNewDecFromStr("-0.3"), resp.Position.LatestCumulativePremiumFraction)
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
		name                string
		initialPosition     types.Position
		pairMetadata        types.PairMetadata
		direction           vpooltypes.Direction
		newPositionNotional sdk.Dec
		quoteAssetLimit     sdk.Dec

		expectedFundingPayment sdk.Dec
		expectedBadDebt        sdk.Dec
		expectedRealizedPnl    sdk.Dec
		expectedMarginToVault  sdk.Dec
	}{
		/*==========================LONG POSITIONS============================*/
		{
			name: "close long position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                    sdk.NewDec(100), // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			pairMetadata: types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			},
			direction:              vpooltypes.Direction_ADD_TO_POOL,
			newPositionNotional:    sdk.NewDec(200),
			quoteAssetLimit:        sdk.NewDec(200),
			expectedFundingPayment: sdk.NewDec(2), // 100 * 0.02
			expectedBadDebt:        sdk.ZeroDec(),
			expectedRealizedPnl:    sdk.NewDec(100),  // 200 - 100
			expectedMarginToVault:  sdk.NewDec(-108), // ( 10(oldMargin) + 100(realzedPnL) - 2(fundingPayment) ) * -1
		},
		{
			name: "close long position, negative PnL",
			// user bought in at 100 BTC for 10.5 NUSD at 10x leverage (1 BTC = 1.05 NUSD)
			// notional value is 105 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -5 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100),               // 100 BTC
				Margin:                          sdk.MustNewDecFromStr("10.5"), // 10.5 NUSD
				OpenNotional:                    sdk.NewDec(105),               // 105 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			pairMetadata: types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			},
			direction:              vpooltypes.Direction_ADD_TO_POOL,
			newPositionNotional:    sdk.NewDec(100),
			quoteAssetLimit:        sdk.NewDec(100),
			expectedFundingPayment: sdk.NewDec(2), // 100 * 0.02
			expectedBadDebt:        sdk.ZeroDec(),
			expectedRealizedPnl:    sdk.NewDec(-5),                // 100 - 105
			expectedMarginToVault:  sdk.MustNewDecFromStr("-3.5"), // ( 10.5(oldMargin) + (-5)(unrealzedPnL) - 2(fundingPayment) ) * -1
		},
		{
			name: "close long position, negative PnL leads to bad debt",
			// user bought in at 100 BTC for 15 NUSD at 10x leverage (1 BTC = 1.5 NUSD)
			// notional value is 150 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -50 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(15),  // 15 NUSD
				OpenNotional:                    sdk.NewDec(150), // 150 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			pairMetadata: types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			},
			direction:              vpooltypes.Direction_ADD_TO_POOL,
			newPositionNotional:    sdk.NewDec(100),
			quoteAssetLimit:        sdk.NewDec(100),
			expectedFundingPayment: sdk.NewDec(2),   // 100 * 0.02
			expectedBadDebt:        sdk.NewDec(37),  // ( 15(oldMargin) + (-50)(unrealzedPnL) - 2(fundingPayment) ) * -1
			expectedRealizedPnl:    sdk.NewDec(-50), // 100 - 150
			expectedMarginToVault:  sdk.ZeroDec(),   // ( 15(oldMargin) + (-50)(unrealzedPnL) - 2(fundingPayment) ) * -1 --> clipped at zero
		},

		/*==========================SHORT POSITIONS===========================*/
		{
			name: "close short position, positive PnL",
			// user bought in at 150 BTC for 15 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 150 NUSD
			// BTC drops in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of +50 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-150), // -150 BTC
				Margin:                          sdk.NewDec(15),   // 15 NUSD
				OpenNotional:                    sdk.NewDec(150),  // 150 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			pairMetadata: types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			},
			direction:              vpooltypes.Direction_REMOVE_FROM_POOL,
			newPositionNotional:    sdk.NewDec(100),
			quoteAssetLimit:        sdk.NewDec(100),
			expectedFundingPayment: sdk.NewDec(-3), // 150 * 0.02
			expectedBadDebt:        sdk.ZeroDec(),
			expectedRealizedPnl:    sdk.NewDec(50),  // 150 - 100
			expectedMarginToVault:  sdk.NewDec(-68), // ( 15(oldMargin) + 50(PnL) - (-3)(fundingPayment) ) * -1
		},
		{
			name: "close short position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1.05 BTC = 1 NUSD
			// user has position notional value of 105 NUSD and unrealized PnL of -5 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			pairMetadata: types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			},
			direction:              vpooltypes.Direction_REMOVE_FROM_POOL,
			newPositionNotional:    sdk.NewDec(105),
			quoteAssetLimit:        sdk.NewDec(105),
			expectedFundingPayment: sdk.NewDec(-2), // 100 * 0.02
			expectedBadDebt:        sdk.ZeroDec(),
			expectedRealizedPnl:    sdk.NewDec(-5), // 150 - 100
			expectedMarginToVault:  sdk.NewDec(-7), // ( 10(oldMargin) + (-5)(PnL) - (-2)(fundingPayment) ) * -1
		},
		{
			name: "close short position, negative PnL leads to bad debt",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 150 NUSD and unrealized PnL of -50 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			pairMetadata: types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			},
			direction:              vpooltypes.Direction_REMOVE_FROM_POOL,
			newPositionNotional:    sdk.NewDec(150),
			quoteAssetLimit:        sdk.NewDec(150),
			expectedFundingPayment: sdk.NewDec(-2),  // 100 * 0.02
			expectedBadDebt:        sdk.NewDec(38),  // 10(oldMargin) + (-50)(PnL) - (-2)(fundingPayment)
			expectedRealizedPnl:    sdk.NewDec(-50), // 100 - 150
			expectedMarginToVault:  sdk.ZeroDec(),   // ( 10(oldMargin) + (-50)(PnL) - (-2)(fundingPayment) ) * -1 --> clipped to zero
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			t.Log("set up initial position")
			setPosition(perpKeeper, ctx, tc.initialPosition)

			t.Log("mock vpool")
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					tc.direction,
					/*baseAssetAmount=*/ tc.initialPosition.Size_.Abs(),
				).
				Return( /*quoteAssetAmount=*/ tc.newPositionNotional, nil)

			mocks.mockVpoolKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					/*quoteAssetDirection=*/ tc.direction,
					/*baseAssetAmount=*/ tc.initialPosition.Size_.Abs(),
					/*quoteAssetLimit=*/ tc.quoteAssetLimit,
					/* skipFluctuationLimitCheck */ false,
				).Return( /*quoteAssetAmount=*/ tc.newPositionNotional, nil)

			t.Log("set up pair metadata and last cumulative funding rate")
			setPairMetadata(perpKeeper, ctx, tc.pairMetadata)

			t.Log("close position")
			resp, err := perpKeeper.closePositionEntirely(
				ctx,
				tc.initialPosition,
				/*quoteAssetLimit=*/ tc.quoteAssetLimit, // NUSD
				/* skipFluctuationLimitCheck */ false,
			)

			require.NoError(t, err)
			assert.EqualValues(t, tc.newPositionNotional, resp.ExchangedNotionalValue)
			assert.EqualValues(t, tc.initialPosition.Size_.Neg(), resp.ExchangedPositionSize) // sold back to vpool
			assert.EqualValues(t, tc.expectedFundingPayment, resp.FundingPayment)
			assert.EqualValues(t, tc.expectedBadDebt, resp.BadDebt)
			assert.EqualValues(t, tc.expectedRealizedPnl, resp.RealizedPnl)
			assert.EqualValues(t, tc.expectedMarginToVault, resp.MarginToVault)
			assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter) // always zero
			assert.Equal(t, sdk.ZeroDec(), resp.PositionNotional)         // always zero

			assert.EqualValues(t, tc.initialPosition.TraderAddress, resp.Position.TraderAddress)
			assert.EqualValues(t, tc.initialPosition.Pair, resp.Position.Pair)
			assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Size_)        // always zero
			assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)       // always zero
			assert.EqualValues(t, sdk.ZeroDec(), resp.Position.OpenNotional) // always zero
			assert.EqualValues(t,
				tc.pairMetadata.LatestCumulativePremiumFraction,
				resp.Position.LatestCumulativePremiumFraction,
			)
			assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
		})
	}
}

func TestDecreasePosition(t *testing.T) {
	tests := []struct {
		name string

		initialPosition       types.Position
		baseAssetDir          vpooltypes.Direction
		quoteAssetDir         vpooltypes.Direction
		priorPositionNotional sdk.Dec
		quoteAmountToDecrease sdk.Dec
		exchangedBaseAmount   sdk.Dec
		baseAssetLimit        sdk.Dec

		expectedFundingPayment     sdk.Dec
		expectedBadDebt            sdk.Dec
		expectedRealizedPnl        sdk.Dec
		expectedUnrealizedPnlAfter sdk.Dec
		expectedPositionNotional   sdk.Dec

		expectedFinalPositionMargin       sdk.Dec
		expectedFinalPositionSize         sdk.Dec
		expectedFinalPositionOpenNotional sdk.Dec
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
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                    sdk.NewDec(100), // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:          vpooltypes.Direction_ADD_TO_POOL,
			priorPositionNotional: sdk.NewDec(200),
			quoteAssetDir:         vpooltypes.Direction_REMOVE_FROM_POOL,
			quoteAmountToDecrease: sdk.NewDec(100),
			exchangedBaseAmount:   sdk.NewDec(-50),
			baseAssetLimit:        sdk.NewDec(50),

			expectedBadDebt:            sdk.ZeroDec(),
			expectedFundingPayment:     sdk.NewDec(2),
			expectedUnrealizedPnlAfter: sdk.NewDec(50),
			expectedRealizedPnl:        sdk.NewDec(50),
			expectedPositionNotional:   sdk.NewDec(100),

			expectedFinalPositionMargin:       sdk.NewDec(58),
			expectedFinalPositionSize:         sdk.NewDec(50),
			expectedFinalPositionOpenNotional: sdk.NewDec(50),
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
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(105),               // 105 BTC
				Margin:                          sdk.MustNewDecFromStr("10.5"), // 10.5 NUSD
				OpenNotional:                    sdk.NewDec(105),               // 105 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:          vpooltypes.Direction_ADD_TO_POOL,
			priorPositionNotional: sdk.NewDec(100),
			quoteAssetDir:         vpooltypes.Direction_REMOVE_FROM_POOL,
			quoteAmountToDecrease: sdk.NewDec(5),
			exchangedBaseAmount:   sdk.MustNewDecFromStr("-5.25"),
			baseAssetLimit:        sdk.MustNewDecFromStr("5.25"),

			expectedBadDebt:            sdk.ZeroDec(),
			expectedFundingPayment:     sdk.MustNewDecFromStr("2.1"),
			expectedUnrealizedPnlAfter: sdk.MustNewDecFromStr("-4.75"),
			expectedRealizedPnl:        sdk.MustNewDecFromStr("-0.25"),
			expectedPositionNotional:   sdk.NewDec(95),

			expectedFinalPositionMargin:       sdk.MustNewDecFromStr("8.15"), // 10.5(old) + (-0.25)(realized PnL) - (2.1)(funding payment)
			expectedFinalPositionSize:         sdk.MustNewDecFromStr("99.75"),
			expectedFinalPositionOpenNotional: sdk.MustNewDecFromStr("99.75"), // 100(position notional) - 5(notional sold) - (-4.75)(unrealized PnL)
		},
		{
			name: "decrease long position, negative PnL, bad debt",
			// user bought in at 100 BTC for 15 NUSD at 10x leverage (1 BTC = 1.5 NUSD)
			// notional value is 150 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -50 NUSD
			// user decreases position by notional value of 50 NUSD
			// user ends up with realized PnL of -25 NUSD, unrealized PnL of -25 NUSD,
			//   position notional value of 50 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(15),  // 15 NUSD
				OpenNotional:                    sdk.NewDec(150), // 150 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:          vpooltypes.Direction_ADD_TO_POOL,
			priorPositionNotional: sdk.NewDec(100),
			quoteAssetDir:         vpooltypes.Direction_REMOVE_FROM_POOL,
			quoteAmountToDecrease: sdk.NewDec(50),
			exchangedBaseAmount:   sdk.NewDec(-50),
			baseAssetLimit:        sdk.NewDec(50),

			expectedBadDebt:            sdk.NewDec(12),
			expectedFundingPayment:     sdk.NewDec(2),
			expectedUnrealizedPnlAfter: sdk.NewDec(-25),
			expectedRealizedPnl:        sdk.NewDec(-25),
			expectedPositionNotional:   sdk.NewDec(50),

			expectedFinalPositionMargin:       sdk.ZeroDec(), // 15(old) + (-25)(realized PnL) - (2)(funding payment)
			expectedFinalPositionSize:         sdk.NewDec(50),
			expectedFinalPositionOpenNotional: sdk.NewDec(75), // 100(position notional) - 50(notional sold) - (-25)(unrealized PnL)
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
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-105),              // -105 BTC
				Margin:                          sdk.MustNewDecFromStr("10.5"), // 10.5 NUSD
				OpenNotional:                    sdk.NewDec(105),               // 105 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:          vpooltypes.Direction_REMOVE_FROM_POOL,
			priorPositionNotional: sdk.NewDec(100),
			quoteAssetDir:         vpooltypes.Direction_ADD_TO_POOL,
			quoteAmountToDecrease: sdk.NewDec(5),
			exchangedBaseAmount:   sdk.MustNewDecFromStr("5.25"),
			baseAssetLimit:        sdk.MustNewDecFromStr("5.25"),

			expectedBadDebt:            sdk.ZeroDec(),
			expectedFundingPayment:     sdk.MustNewDecFromStr("-2.1"),
			expectedUnrealizedPnlAfter: sdk.MustNewDecFromStr("4.75"),
			expectedRealizedPnl:        sdk.MustNewDecFromStr("0.25"),
			expectedPositionNotional:   sdk.NewDec(95),

			expectedFinalPositionMargin:       sdk.MustNewDecFromStr("12.85"), // old(10.5) + (0.25)(realizedPnL) - (-2.1)(fundingPayment)
			expectedFinalPositionSize:         sdk.MustNewDecFromStr("-99.75"),
			expectedFinalPositionOpenNotional: sdk.MustNewDecFromStr("99.75"),
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
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:          vpooltypes.Direction_REMOVE_FROM_POOL,
			priorPositionNotional: sdk.NewDec(105),
			quoteAssetDir:         vpooltypes.Direction_ADD_TO_POOL,
			quoteAmountToDecrease: sdk.MustNewDecFromStr("5.25"),
			exchangedBaseAmount:   sdk.NewDec(5),
			baseAssetLimit:        sdk.NewDec(5),

			expectedBadDebt:            sdk.ZeroDec(),
			expectedFundingPayment:     sdk.NewDec(-2),
			expectedUnrealizedPnlAfter: sdk.MustNewDecFromStr("-4.75"),
			expectedRealizedPnl:        sdk.MustNewDecFromStr("-0.25"),
			expectedPositionNotional:   sdk.MustNewDecFromStr("99.75"),

			expectedFinalPositionMargin:       sdk.MustNewDecFromStr("11.75"), // old(10) + (-0.25)(realizedPnL) - (-2)(fundingPayment)
			expectedFinalPositionSize:         sdk.NewDec(-95),
			expectedFinalPositionOpenNotional: sdk.NewDec(95),
		},
		{
			name: "decrease short position, negative PnL, bad debt",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1 BTC = 1.5 NUSD
			// user has position notional value of 150 NUSD and unrealized PnL of -50 NUSD
			// user decreases position by notional value of 75 NUSD
			// user ends up with realized PnL of -25 NUSD, unrealized PnL of -25 NUSD
			//   position notional value of 75 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:          vpooltypes.Direction_REMOVE_FROM_POOL,
			priorPositionNotional: sdk.NewDec(150),
			quoteAssetDir:         vpooltypes.Direction_ADD_TO_POOL,
			quoteAmountToDecrease: sdk.NewDec(75),
			exchangedBaseAmount:   sdk.NewDec(50),
			baseAssetLimit:        sdk.NewDec(50),

			expectedBadDebt:            sdk.NewDec(13), // old(10) + (-25)(realizedPnL) - (-2)(fundingPayment)
			expectedFundingPayment:     sdk.NewDec(-2),
			expectedUnrealizedPnlAfter: sdk.NewDec(-25),
			expectedRealizedPnl:        sdk.NewDec(-25),
			expectedPositionNotional:   sdk.NewDec(75),

			expectedFinalPositionMargin:       sdk.ZeroDec(),
			expectedFinalPositionSize:         sdk.NewDec(-50),
			expectedFinalPositionOpenNotional: sdk.NewDec(50),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			t.Log("mock vpool")
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					tc.baseAssetDir,
					/*baseAssetAmount=*/ tc.initialPosition.Size_.Abs(),
				).
				Return( /*quoteAssetAmount=*/ tc.priorPositionNotional, nil)

			mocks.mockVpoolKeeper.EXPECT().
				SwapQuoteForBase(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					/*quoteAssetDirection=*/ tc.quoteAssetDir,
					/*quoteAssetAmount=*/ tc.quoteAmountToDecrease,
					/*baseAssetLimit=*/ tc.exchangedBaseAmount.Abs(),
					/* skipFluctuationLimitCheck */ false,
				).Return( /*baseAssetAmount=*/ tc.baseAssetLimit, nil)

			t.Log("set up pair metadata and last cumulative funding rate")
			setPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			})

			t.Log("decrease position")
			resp, err := perpKeeper.decreasePosition(
				ctx,
				tc.initialPosition,
				/*openNotional=*/ tc.quoteAmountToDecrease, // NUSD
				/*baseLimit=*/ tc.baseAssetLimit, // BTC
				/*skipFluctuationLimitCheck=*/ false,
			)

			require.NoError(t, err)
			assert.EqualValues(t, tc.quoteAmountToDecrease, resp.ExchangedNotionalValue)
			assert.EqualValues(t, tc.expectedBadDebt, resp.BadDebt)
			assert.EqualValues(t, tc.exchangedBaseAmount, resp.ExchangedPositionSize)
			assert.EqualValues(t, tc.expectedFundingPayment, resp.FundingPayment)
			assert.EqualValues(t, tc.expectedRealizedPnl, resp.RealizedPnl)
			assert.EqualValues(t, tc.expectedUnrealizedPnlAfter, resp.UnrealizedPnlAfter)
			assert.EqualValues(t, tc.expectedPositionNotional, resp.PositionNotional)
			assert.EqualValues(t, sdk.ZeroDec(), resp.MarginToVault) // always zero for DecreasePosition

			assert.EqualValues(t, tc.initialPosition.TraderAddress, resp.Position.TraderAddress)
			assert.EqualValues(t, tc.initialPosition.Pair, resp.Position.Pair)
			assert.EqualValues(t, tc.expectedFinalPositionSize, resp.Position.Size_)
			assert.EqualValues(t, tc.expectedFinalPositionMargin, resp.Position.Margin)
			assert.EqualValues(t, tc.expectedFinalPositionOpenNotional, resp.Position.OpenNotional)
			assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LatestCumulativePremiumFraction)
			assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
		})
	}
}

func TestCloseAndOpenReversePosition(t *testing.T) {
	tests := []struct {
		name string

		initialPositionSize         sdk.Dec
		initialPositionMargin       sdk.Dec
		initialPositionOpenNotional sdk.Dec

		mockBaseDir     vpooltypes.Direction
		mockQuoteAmount sdk.Dec
		mockQuoteDir    vpooltypes.Direction
		mockBaseAmount  sdk.Dec

		inputQuoteAmount    sdk.Dec
		inputLeverage       sdk.Dec
		inputBaseAssetLimit sdk.Dec

		expectedPositionResp types.PositionResp

		expectedErr error
	}{
		/*==========================LONG POSITIONS============================*/
		{
			name: "close long position, positive PnL, open short position",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user closes position and opens in reverse direction with 30*10 NUSD
			initialPositionSize:         sdk.NewDec(100),
			initialPositionMargin:       sdk.NewDec(10),
			initialPositionOpenNotional: sdk.NewDec(100),

			mockBaseDir:     vpooltypes.Direction_ADD_TO_POOL,
			mockQuoteAmount: sdk.NewDec(200),
			mockQuoteDir:    vpooltypes.Direction_REMOVE_FROM_POOL,
			mockBaseAmount:  sdk.NewDec(50),

			inputQuoteAmount:    sdk.NewDec(30),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.NewDec(150),

			expectedPositionResp: types.PositionResp{
				ExchangedNotionalValue: sdk.NewDec(300),  // 30 * 10
				ExchangedPositionSize:  sdk.NewDec(-150), // 100 original + 50 shorted
				PositionNotional:       sdk.NewDec(100),  // abs(200 - 300)
				MarginToVault:          sdk.NewDec(-98),  // -1 * ( 10(oldMargin) + 100(unrealzedPnL) - 2(fundingPayment) ) + 10
				RealizedPnl:            sdk.NewDec(100),  // 200 - 100
				UnrealizedPnlAfter:     sdk.ZeroDec(),    // always zero
				FundingPayment:         sdk.NewDec(2),    // 100 * 0.02
				BadDebt:                sdk.ZeroDec(),
				Position: &types.Position{
					Size_:        sdk.NewDec(-50),
					Margin:       sdk.NewDec(10),
					OpenNotional: sdk.NewDec(100),
				},
			},
		},
		{
			name: "close long position, negative PnL, open short position",
			// user bought in at 100 BTC for 10.5 NUSD at 10x leverage (1 BTC = 1.05 NUSD)
			// notional value is 105 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -5 NUSD
			// user closes position and opens in reverse direction with 30*10 NUSD
			initialPositionSize:         sdk.NewDec(100),
			initialPositionMargin:       sdk.NewDec(11),
			initialPositionOpenNotional: sdk.NewDec(105),

			mockBaseDir:     vpooltypes.Direction_ADD_TO_POOL,
			mockQuoteAmount: sdk.NewDec(100),
			mockQuoteDir:    vpooltypes.Direction_REMOVE_FROM_POOL,
			mockBaseAmount:  sdk.NewDec(100),

			inputQuoteAmount:    sdk.NewDec(20),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.NewDec(200),

			expectedPositionResp: types.PositionResp{
				ExchangedNotionalValue: sdk.NewDec(200),  // 20 * 10
				ExchangedPositionSize:  sdk.NewDec(-200), // 100 original + 100 shorted
				PositionNotional:       sdk.NewDec(100),  // abs(200 - 300)
				MarginToVault:          sdk.NewDec(6),    // -1 * ( 11(oldMargin) + (-5)(unrealzedPnL) - 2(fundingPayment) ) + 10
				RealizedPnl:            sdk.NewDec(-5),   // 100 - 105
				UnrealizedPnlAfter:     sdk.ZeroDec(),    // always zero
				FundingPayment:         sdk.NewDec(2),    // 100 * 0.02
				BadDebt:                sdk.ZeroDec(),
				Position: &types.Position{
					Size_:        sdk.NewDec(-100),
					Margin:       sdk.NewDec(10),
					OpenNotional: sdk.NewDec(100),
				},
			},
		},
		{
			name: "close long position, negative PnL leads to bad debt, cannot close and open reverse",
			// user bought in at 100 BTC for 15 NUSD at 10x leverage (1 BTC = 1.5 NUSD)
			// notional value is 150 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -50 NUSD
			// user tries to close and open reverse position but cannot because it leads to bad debt
			initialPositionSize:         sdk.NewDec(100),
			initialPositionMargin:       sdk.NewDec(15),
			initialPositionOpenNotional: sdk.NewDec(150),

			mockBaseDir:     vpooltypes.Direction_ADD_TO_POOL,
			mockQuoteAmount: sdk.NewDec(100),

			inputQuoteAmount:    sdk.NewDec(20),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.NewDec(200),

			expectedErr: fmt.Errorf("underwater position"),
		},
		{
			name: "existing long position, positive PnL, zero base asset limit",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user closes position and opens in reverse direction with 30*10 NUSD
			initialPositionSize:         sdk.NewDec(100),
			initialPositionMargin:       sdk.NewDec(10),
			initialPositionOpenNotional: sdk.NewDec(100),

			mockBaseDir:     vpooltypes.Direction_ADD_TO_POOL,
			mockQuoteAmount: sdk.NewDec(200),
			mockQuoteDir:    vpooltypes.Direction_REMOVE_FROM_POOL,
			mockBaseAmount:  sdk.NewDec(50),

			inputQuoteAmount:    sdk.NewDec(30),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.ZeroDec(),

			expectedPositionResp: types.PositionResp{
				ExchangedNotionalValue: sdk.NewDec(300),  // 30 * 10
				ExchangedPositionSize:  sdk.NewDec(-150), // 100 original + 50 shorted
				PositionNotional:       sdk.NewDec(100),  // abs(200 - 300)
				MarginToVault:          sdk.NewDec(-98),  // -1 * ( 10(oldMargin) + 100(unrealzedPnL) - 2(fundingPayment) ) + 10
				RealizedPnl:            sdk.NewDec(100),  // 200 - 100
				UnrealizedPnlAfter:     sdk.ZeroDec(),    // always zero
				FundingPayment:         sdk.NewDec(2),    // 100 * 0.02
				BadDebt:                sdk.ZeroDec(),
				Position: &types.Position{
					Size_:        sdk.NewDec(-50),
					Margin:       sdk.NewDec(10),
					OpenNotional: sdk.NewDec(100),
				},
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
			initialPositionSize:         sdk.NewDec(100),
			initialPositionMargin:       sdk.NewDec(10),
			initialPositionOpenNotional: sdk.NewDec(100),

			mockBaseDir:     vpooltypes.Direction_ADD_TO_POOL,
			mockQuoteAmount: sdk.NewDec(200),

			inputQuoteAmount:    sdk.NewDec(30),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.NewDec(5),

			expectedErr: fmt.Errorf("position size changed by greater than the specified base limit"),
		},

		/*==========================SHORT POSITIONS===========================*/
		{
			name: "close short position, positive PnL",
			// user opened position at 150 BTC for 15 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 150 NUSD
			// BTC drops in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of +50 NUSD
			// user closes and opens position in reverse with 20*10 notional value
			initialPositionSize:         sdk.NewDec(-150),
			initialPositionMargin:       sdk.NewDec(15),
			initialPositionOpenNotional: sdk.NewDec(150),

			mockBaseDir:     vpooltypes.Direction_REMOVE_FROM_POOL,
			mockQuoteAmount: sdk.NewDec(100),
			mockQuoteDir:    vpooltypes.Direction_ADD_TO_POOL,
			mockBaseAmount:  sdk.NewDec(150),

			inputQuoteAmount:    sdk.NewDec(20),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.NewDec(300),

			expectedPositionResp: types.PositionResp{
				ExchangedNotionalValue: sdk.NewDec(200), // 20 * 10
				ExchangedPositionSize:  sdk.NewDec(300), // 150 original + 150 long
				PositionNotional:       sdk.NewDec(100), // abs(100 - 200)
				MarginToVault:          sdk.NewDec(-58), // -1 * ( 15(oldMargin) + 50(unrealzedPnL) - (-3)(fundingPayment) ) + 10
				RealizedPnl:            sdk.NewDec(50),  // 150 - 100
				UnrealizedPnlAfter:     sdk.ZeroDec(),   // always zero
				FundingPayment:         sdk.NewDec(-3),  // -150 * 0.02
				BadDebt:                sdk.ZeroDec(),
				Position: &types.Position{
					Size_:        sdk.NewDec(150),
					Margin:       sdk.NewDec(10),
					OpenNotional: sdk.NewDec(100),
				},
			},
		},
		{
			name: "close short position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1.05 BTC = 1 NUSD
			// user has position notional value of 105 NUSD and unrealized PnL of -5 NUSD
			// user closes and opens reverse with 21 * 10 notional value
			initialPositionSize:         sdk.NewDec(-100),
			initialPositionMargin:       sdk.NewDec(10),
			initialPositionOpenNotional: sdk.NewDec(100),

			mockBaseDir:     vpooltypes.Direction_REMOVE_FROM_POOL,
			mockQuoteAmount: sdk.NewDec(105),
			mockQuoteDir:    vpooltypes.Direction_ADD_TO_POOL,
			mockBaseAmount:  sdk.NewDec(100),

			inputQuoteAmount:    sdk.NewDec(21),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.NewDec(200),

			expectedPositionResp: types.PositionResp{
				ExchangedNotionalValue: sdk.NewDec(210),              // 21 * 10
				ExchangedPositionSize:  sdk.NewDec(200),              // 100 original + 100 long
				PositionNotional:       sdk.NewDec(105),              // abs(105 - 210)
				MarginToVault:          sdk.MustNewDecFromStr("3.5"), // -1 * ( 10(oldMargin) + (-5)(unrealzedPnL) - (-2)(fundingPayment) ) + 10.5
				RealizedPnl:            sdk.NewDec(-5),               // 100 - 105
				UnrealizedPnlAfter:     sdk.ZeroDec(),                // always zero
				FundingPayment:         sdk.NewDec(-2),               // -100 * 0.02
				BadDebt:                sdk.ZeroDec(),
				Position: &types.Position{
					Size_:        sdk.NewDec(100),
					Margin:       sdk.MustNewDecFromStr("10.5"),
					OpenNotional: sdk.NewDec(105),
				},
			},
		},
		{
			name: "close short position, negative PnL leads to bad debt",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 150 NUSD and unrealized PnL of -50 NUSD
			// user tries to close and open reverse position but cannot due to being underwater
			initialPositionSize:         sdk.NewDec(-100),
			initialPositionMargin:       sdk.NewDec(10),
			initialPositionOpenNotional: sdk.NewDec(100),

			mockBaseDir:     vpooltypes.Direction_REMOVE_FROM_POOL,
			mockQuoteAmount: sdk.NewDec(150),

			inputQuoteAmount:    sdk.NewDec(21),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.NewDec(200),

			expectedErr: fmt.Errorf("underwater position"),
		},
		{
			name: "close short position, positive PnL, no base amount limit",
			// user opened position at 150 BTC for 15 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 150 NUSD
			// BTC drops in value, now its price is 1.5 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of +50 NUSD
			// user closes and opens position in reverse with 20*10 notional value
			initialPositionSize:         sdk.NewDec(-150),
			initialPositionMargin:       sdk.NewDec(15),
			initialPositionOpenNotional: sdk.NewDec(150),

			mockBaseDir:     vpooltypes.Direction_REMOVE_FROM_POOL,
			mockQuoteAmount: sdk.NewDec(100),

			mockQuoteDir:   vpooltypes.Direction_ADD_TO_POOL,
			mockBaseAmount: sdk.NewDec(150),

			inputQuoteAmount:    sdk.NewDec(20),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.ZeroDec(),

			expectedPositionResp: types.PositionResp{
				ExchangedNotionalValue: sdk.NewDec(200), // 20 * 10
				ExchangedPositionSize:  sdk.NewDec(300), // 150 original + 150 long
				PositionNotional:       sdk.NewDec(100), // abs(100 - 200)
				MarginToVault:          sdk.NewDec(-58), // -1 * ( 15(oldMargin) + 50(PnL) - (-3)(fundingPayment) ) + 10
				RealizedPnl:            sdk.NewDec(50),  // 150 - 100
				UnrealizedPnlAfter:     sdk.ZeroDec(),   // always zero
				FundingPayment:         sdk.NewDec(-3),  // -150 * 0.02
				BadDebt:                sdk.ZeroDec(),
				Position: &types.Position{
					Size_:        sdk.NewDec(150),
					Margin:       sdk.NewDec(10),
					OpenNotional: sdk.NewDec(100),
				},
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
			initialPositionSize:         sdk.NewDec(-150),
			initialPositionMargin:       sdk.NewDec(15),
			initialPositionOpenNotional: sdk.NewDec(150),

			mockBaseDir:     vpooltypes.Direction_REMOVE_FROM_POOL,
			mockQuoteAmount: sdk.NewDec(100),

			inputQuoteAmount:    sdk.NewDec(20),
			inputLeverage:       sdk.NewDec(10),
			inputBaseAssetLimit: sdk.NewDec(5),

			expectedErr: fmt.Errorf("position size changed by greater than the specified base limit"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			traderAddr := testutilevents.AccAddress()

			t.Log("set up initial position")
			currentPosition := types.Position{
				TraderAddress:                   traderAddr.String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           tc.initialPositionSize,
				Margin:                          tc.initialPositionMargin,
				OpenNotional:                    tc.initialPositionOpenNotional,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			}
			setPosition(perpKeeper, ctx, currentPosition)

			t.Log("mock vpool")
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					tc.mockBaseDir,
					/*baseAssetAmount=*/ currentPosition.Size_.Abs(),
				).
				Return( /*quoteAssetAmount=*/ tc.mockQuoteAmount, nil)

			mocks.mockVpoolKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					tc.mockBaseDir,
					/*baseAssetAmount=*/ currentPosition.Size_.Abs(),
					/*quoteAssetLimit=*/ sdk.ZeroDec(),
					/* skipFluctuationLimitCheck */ false,
				).Return( /*quoteAssetAmount=*/ tc.mockQuoteAmount, nil)

			if tc.expectedErr == nil {
				mocks.mockVpoolKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
						/*quoteAssetDirection=*/ tc.mockQuoteDir,
						/*quoteAssetAmount=*/ tc.inputQuoteAmount.Mul(tc.inputLeverage).Sub(tc.mockQuoteAmount),
						/*baseAssetLimit=*/ sdk.MaxDec(tc.inputBaseAssetLimit.Sub(currentPosition.Size_.Abs()), sdk.ZeroDec()),
						/* skipFluctuationLimitCheck */ false,
					).Return( /*baseAssetAmount=*/ tc.mockBaseAmount, nil)
			}

			t.Log("set up pair metadata and last cumulative funding rate")
			setPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			})

			t.Log("close position and open reverse")
			resp, err := perpKeeper.closeAndOpenReversePosition(
				ctx,
				currentPosition,
				/*quoteAssetAmount=*/ tc.inputQuoteAmount, // NUSD
				/*leverage=*/ tc.inputLeverage,
				/*baseAssetLimit=*/ tc.inputBaseAssetLimit,
			)

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.EqualValues(t, tc.expectedPositionResp.ExchangedNotionalValue, resp.ExchangedNotionalValue)
				assert.EqualValues(t, tc.expectedPositionResp.ExchangedPositionSize, resp.ExchangedPositionSize)
				assert.EqualValues(t, tc.expectedPositionResp.FundingPayment, resp.FundingPayment)
				assert.EqualValues(t, tc.expectedPositionResp.MarginToVault, resp.MarginToVault)
				assert.EqualValues(t, tc.expectedPositionResp.PositionNotional, resp.PositionNotional)
				assert.EqualValues(t, tc.expectedPositionResp.BadDebt, resp.BadDebt)
				assert.EqualValues(t, tc.expectedPositionResp.RealizedPnl, resp.RealizedPnl)
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter) // always zero

				assert.EqualValues(t, currentPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, currentPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, tc.expectedPositionResp.Position.Size_, resp.Position.Size_)
				assert.EqualValues(t, tc.expectedPositionResp.Position.Margin, resp.Position.Margin)
				assert.EqualValues(t, tc.expectedPositionResp.Position.OpenNotional, resp.Position.OpenNotional)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LatestCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			}
		})
	}
}

func TestTransferFee(t *testing.T) {
	setup := func() (
		k Keeper, mocks mockedDependencies, ctx sdk.Context, pair common.AssetPair,
		trader sdk.AccAddress, positionNotional sdk.Dec,
	) {
		perpKeeper, mocks, ctx := getKeeper(t)
		pair = common.MustNewAssetPair("btc:usdc")
		perpKeeper.SetParams(ctx, types.DefaultParams())
		metadata := &types.PairMetadata{
			Pair: pair,
		}
		setPairMetadata(perpKeeper, ctx, *metadata)
		trader = testutilevents.AccAddress()
		positionNotional = sdk.NewDec(5_000)
		return perpKeeper, mocks, ctx, pair, trader, positionNotional
	}

	t.Run("trader has funds - happy",
		func(t *testing.T) {
			k, mocks, ctx, pair, trader, positionNotional := setup()

			var wantError error = nil
			mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
				ctx, trader, types.FeePoolModuleAccount,
				sdk.NewCoins(sdk.NewInt64Coin(pair.QuoteDenom(), 5)),
			).Return(wantError)
			mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
				ctx, trader, types.PerpEFModuleAccount,
				sdk.NewCoins(sdk.NewInt64Coin(pair.QuoteDenom(), 5)),
			).Return(wantError)

			_, err := k.transferFee(
				ctx, pair, trader, positionNotional)
			require.NoError(t, err)
		})

	t.Run("not enough funds for Perp Ecosystem Fund (spread) - error",
		func(t *testing.T) {
			k, mocks, ctx, pair, trader, positionNotional := setup()

			mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
				ctx, trader, types.FeePoolModuleAccount,
				sdk.NewCoins(sdk.NewInt64Coin(pair.QuoteDenom(), 5)),
			).Return(nil)

			expectedError := fmt.Errorf(
				"trader missing funds for %s", types.PerpEFModuleAccount)
			mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
				ctx, trader, types.PerpEFModuleAccount,
				sdk.NewCoins(sdk.NewInt64Coin(pair.QuoteDenom(), 5))).
				Return(expectedError)
			_, err := k.transferFee(
				ctx, pair, trader, positionNotional)
			require.ErrorContains(t, err, expectedError.Error())
		})

	t.Run("not enough funds for Fee Pool (toll) - error",
		func(t *testing.T) {
			k, mocks, ctx, pair, trader, positionNotional := setup()

			expectedError := fmt.Errorf(
				"trader missing funds for %s", types.FeePoolModuleAccount)
			mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
				ctx,
				/* from */ trader,
				/* to */ types.FeePoolModuleAccount,
				sdk.NewCoins(sdk.NewInt64Coin(pair.QuoteDenom(), 5)),
			).Return(expectedError)

			_, err := k.transferFee(
				ctx, pair, trader, positionNotional)
			require.ErrorContains(t, err, expectedError.Error())
		})
}

func TestClosePosition(t *testing.T) {
	tests := []struct {
		name string

		initialPosition     types.Position
		baseAssetDir        vpooltypes.Direction
		newPositionNotional sdk.Dec

		expectedFundingPayment sdk.Dec
		expectedBadDebt        sdk.Dec
		expectedRealizedPnl    sdk.Dec
		expectedMarginToVault  sdk.Dec
	}{
		{
			name: "long position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user closes position
			// user ends up with realized PnL of +100 NUSD, unrealized PnL after of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                    sdk.NewDec(100), // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:        vpooltypes.Direction_ADD_TO_POOL,
			newPositionNotional: sdk.NewDec(200),

			expectedBadDebt:        sdk.ZeroDec(),
			expectedFundingPayment: sdk.NewDec(2),
			expectedRealizedPnl:    sdk.NewDec(100),
			expectedMarginToVault:  sdk.NewDec(-108),
		},
		{
			name: "close long position, negative PnL",
			// user bought in at 105 BTC for 10.5 NUSD at 10x leverage (1 BTC = 1 NUSD)
			//   position and open notional value is 105 NUSD
			// BTC drops in value, now its price is 1.05 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -5 NUSD
			// user closes position
			// user ends up with realized PnL of -5 NUSD, unrealized PnL of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(105),               // 105 BTC
				Margin:                          sdk.MustNewDecFromStr("10.5"), // 10.5 NUSD
				OpenNotional:                    sdk.NewDec(105),               // 105 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:        vpooltypes.Direction_ADD_TO_POOL,
			newPositionNotional: sdk.NewDec(100),

			expectedBadDebt:        sdk.ZeroDec(),
			expectedFundingPayment: sdk.MustNewDecFromStr("2.1"),
			expectedRealizedPnl:    sdk.NewDec(-5),
			expectedMarginToVault:  sdk.MustNewDecFromStr("-3.4"), // 10.5(old) + (-5)(realized PnL) - (2.1)(funding payment)
		},

		/*==========================SHORT POSITIONS===========================*/
		{
			name: "close short position, positive PnL",
			// user bought in at 105 BTC for 10.5 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 105 NUSD
			// BTC drops in value, now its price is 1.05 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of 5 NUSD
			// user closes position
			// user ends up with realized PnL of 5 NUSD, unrealized PnL of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-105),              // -105 BTC
				Margin:                          sdk.MustNewDecFromStr("10.5"), // 10.5 NUSD
				OpenNotional:                    sdk.NewDec(105),               // 105 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:        vpooltypes.Direction_REMOVE_FROM_POOL,
			newPositionNotional: sdk.NewDec(100),

			expectedBadDebt:        sdk.ZeroDec(),
			expectedFundingPayment: sdk.MustNewDecFromStr("-2.1"),
			expectedRealizedPnl:    sdk.NewDec(5),
			expectedMarginToVault:  sdk.MustNewDecFromStr("-17.6"), // old(10.5) + (5)(realizedPnL) - (-2.1)(fundingPayment)
		},
		{
			name: "decrease short position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1 BTC = 1.05 NUSD
			// user has position notional value of 105 NUSD and unrealized PnL of -5 NUSD
			// user closes their position
			// user ends up with realized PnL of -5 NUSD, unrealized PnL of 0 NUSD
			//   position notional value of 0 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:        vpooltypes.Direction_REMOVE_FROM_POOL,
			newPositionNotional: sdk.NewDec(105),

			expectedBadDebt:        sdk.ZeroDec(),
			expectedFundingPayment: sdk.NewDec(-2),
			expectedRealizedPnl:    sdk.NewDec(-5),
			expectedMarginToVault:  sdk.NewDec(-7), // old(10) + (-0.25)(realizedPnL) - (-2)(fundingPayment)
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			traderAddr, err := sdk.AccAddressFromBech32(tc.initialPosition.TraderAddress)
			require.NoError(t, err)

			t.Log("set position")
			setPosition(perpKeeper, ctx, tc.initialPosition)

			t.Log("set params")
			params := types.DefaultParams()
			params.FeePoolFeeRatio = sdk.ZeroDec()
			params.EcosystemFundFeeRatio = sdk.ZeroDec()
			perpKeeper.SetParams(ctx, params)

			t.Log("mock vpool keeper")
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					tc.baseAssetDir,
					/*baseAssetAmount=*/ tc.initialPosition.Size_.Abs(),
				).
				Return( /*quoteAssetAmount=*/ tc.newPositionNotional, nil)

			mocks.mockVpoolKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					/*baseAssetDirection=*/ tc.baseAssetDir,
					/*baseAssetAmount=*/ tc.initialPosition.Size_.Abs(),
					/*quoteAssetLimit=*/ sdk.ZeroDec(),
					/* skipFluctuationLimitCheck */ false,
				).Return( /*quoteAssetAmount=*/ tc.newPositionNotional, nil)

			mocks.mockVpoolKeeper.EXPECT().
				GetMarkPrice(
					ctx,
					tc.initialPosition.Pair,
				).Return(
				tc.newPositionNotional.Quo(tc.initialPosition.Size_.Abs()),
				nil,
			)

			t.Log("mock bank keeper")
			t.Logf("expecting sending: %s", sdk.NewCoin(tc.initialPosition.Pair.QuoteDenom(), tc.expectedMarginToVault.RoundInt().Abs()))
			mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
				ctx,
				types.VaultModuleAccount,
				traderAddr,
				sdk.NewCoins(
					sdk.NewCoin(
						/* NUSD */ tc.initialPosition.Pair.QuoteDenom(),
						tc.expectedMarginToVault.RoundInt().Abs(),
					),
				),
			).Return(nil)

			mocks.mockBankKeeper.EXPECT().GetBalance(ctx, sdk.AccAddress{0x1, 0x2, 0x3}, tc.initialPosition.Pair.QuoteDenom()).
				Return(sdk.NewCoin("NUSD", sdk.NewInt(100000*common.Precision)))
			mocks.mockAccountKeeper.EXPECT().GetModuleAddress(types.VaultModuleAccount).
				Return(sdk.AccAddress{0x1, 0x2, 0x3})

			t.Log("set up pair metadata and last cumulative funding rate")
			setPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			})

			t.Log("close position")
			resp, err := perpKeeper.ClosePosition(
				ctx,
				tc.initialPosition.Pair,
				traderAddr,
			)

			require.NoError(t, err)
			assert.EqualValues(t, tc.newPositionNotional, resp.ExchangedNotionalValue)
			assert.EqualValues(t, tc.expectedBadDebt, resp.BadDebt)
			assert.EqualValues(t, tc.initialPosition.Size_.Neg(), resp.ExchangedPositionSize)
			assert.EqualValues(t, tc.expectedFundingPayment, resp.FundingPayment)
			assert.EqualValues(t, tc.expectedRealizedPnl, resp.RealizedPnl)
			assert.EqualValues(t, tc.expectedMarginToVault, resp.MarginToVault)
			assert.EqualValues(t, sdk.ZeroDec(), resp.PositionNotional)   // always zero
			assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter) // always zero

			assert.EqualValues(t, tc.initialPosition.TraderAddress, resp.Position.TraderAddress)
			assert.EqualValues(t, tc.initialPosition.Pair, resp.Position.Pair)
			assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Margin)       // alwayz zero
			assert.EqualValues(t, sdk.ZeroDec(), resp.Position.OpenNotional) // always zero
			assert.EqualValues(t, sdk.ZeroDec(), resp.Position.Size_)        // always zero
			assert.EqualValues(t, sdk.MustNewDecFromStr("0.02"), resp.Position.LatestCumulativePremiumFraction)
			assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionChangedEvent{
				Pair:               tc.initialPosition.Pair,
				TraderAddress:      tc.initialPosition.TraderAddress,
				Margin:             sdk.NewInt64Coin(tc.initialPosition.Pair.QuoteDenom(), 0),
				PositionNotional:   sdk.ZeroDec(),
				ExchangedNotional:  tc.newPositionNotional,
				ExchangedSize:      tc.initialPosition.Size_.Neg(),
				PositionSize:       sdk.ZeroDec(),
				RealizedPnl:        tc.expectedRealizedPnl,
				UnrealizedPnlAfter: sdk.ZeroDec(),
				BadDebt:            sdk.NewCoin(common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD).QuoteDenom(), sdk.ZeroInt()),
				MarkPrice:          tc.newPositionNotional.Quo(tc.initialPosition.Size_.Abs()),
				FundingPayment:     sdk.MustNewDecFromStr("0.02").Mul(tc.initialPosition.Size_),
				TransactionFee:     sdk.NewInt64Coin(tc.initialPosition.Pair.QuoteDenom(), 0),
				BlockHeight:        ctx.BlockHeight(),
				BlockTimeMs:        ctx.BlockTime().UnixMilli(),
			})
		})
	}
}

func TestClosePositionWithBadDebt(t *testing.T) {
	tests := []struct {
		name string

		initialPosition     types.Position
		baseAssetDir        vpooltypes.Direction
		newPositionNotional sdk.Dec
	}{
		{
			name: "close long position, negative PnL, bad debt",
			// user bought in at 100 BTC for 15 NUSD at 10x leverage (1 BTC = 1.5 NUSD)
			//   notional value is 150 NUSD
			// BTC drops in value, now its price is 1 BTC = 1 NUSD
			// user has position notional value of 100 NUSD and unrealized PnL of -50 NUSD
			// user cannot close position due to bad debt
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(15),  // 15 NUSD
				OpenNotional:                    sdk.NewDec(150), // 150 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:        vpooltypes.Direction_ADD_TO_POOL,
			newPositionNotional: sdk.NewDec(100),
		},

		/*==========================SHORT POSITIONS===========================*/
		{
			name: "decrease short position, negative PnL, bad debt",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1 BTC = 1.5 NUSD
			// user has position notional value of 150 NUSD and unrealized PnL of -50 NUSD
			// user cannot close position due to bad debt
			initialPosition: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     0,
			},
			baseAssetDir:        vpooltypes.Direction_REMOVE_FROM_POOL,
			newPositionNotional: sdk.NewDec(150),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			traderAddr, err := sdk.AccAddressFromBech32(tc.initialPosition.TraderAddress)
			require.NoError(t, err)

			t.Log("set position")
			setPosition(perpKeeper, ctx, tc.initialPosition)

			t.Log("set params")
			perpKeeper.SetParams(ctx, types.DefaultParams())

			t.Log("mock vpool keeper")
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					tc.baseAssetDir,
					/*baseAssetAmount=*/ tc.initialPosition.Size_.Abs(),
				).
				AnyTimes().
				Return( /*quoteAssetAmount=*/ tc.newPositionNotional, nil)

			mocks.mockVpoolKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
					/*baseAssetDirection=*/ tc.baseAssetDir,
					/*baseAssetAmount=*/ tc.initialPosition.Size_.Abs(),
					/*quoteAssetLimit=*/ sdk.ZeroDec(),
					/* skipFluctuationLimitCheck */ false,
				).Return( /*quoteAssetAmount=*/ tc.newPositionNotional, nil)

			t.Log("set up pair metadata and last cumulative funding rate")
			setPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            common.AssetRegistry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.02"),
			})

			t.Log("close position")
			resp, err := perpKeeper.ClosePosition(
				ctx,
				tc.initialPosition.Pair,
				traderAddr,
			)

			require.ErrorContains(t, err, "underwater position")
			require.Nil(t, resp)
		})
	}
}

func setPosition(k Keeper, ctx sdk.Context, pos types.Position) {
	k.Positions.Insert(ctx, collections.Join(pos.Pair, sdk.MustAccAddressFromBech32(pos.TraderAddress)), pos)
}

func setPairMetadata(k Keeper, ctx sdk.Context, pm types.PairMetadata) {
	k.PairsMetadata.Insert(ctx, pm.Pair, pm)
}
