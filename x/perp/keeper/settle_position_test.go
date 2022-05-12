package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
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
	"testing"
)

func TestKeeper_SettlePosition(t *testing.T) {
	t.Run("success - settlement price zero", func(t *testing.T) {
		k, dep, ctx := getKeeper(t)
		addr := sample.AccAddress()
		pair, err := common.NewTokenPairFromStr("LUNA:UST") // memeing
		require.NoError(t, err)

		dep.mockVpoolKeeper.
			EXPECT().
			GetSettlementPrice(gomock.Eq(ctx), gomock.Eq(pair)).
			Return(sdk.ZeroDec(), error(nil))

		pos := &types.Position{
			Address:      addr.String(),
			Pair:         pair.String(),
			Size_:        sdk.NewDec(100),
			Margin:       sdk.NewDec(100),
			OpenNotional: sdk.NewDec(1000),
		}
		err = k.Positions().Create(ctx, pos)
		require.NoError(t, err)

		coins, err := k.SettlePosition(ctx, pair, addr.String())
		require.NoError(t, err)

		require.Equal(t, sdk.NewCoins(sdk.NewCoin("todo", pos.Size_.RoundInt())), coins) // TODO(mercilex): here we should have different denom, depends on Transfer impl
	})

	t.Run("success - settlement price not zero", func(t *testing.T) {
		k, dep, ctx := getKeeper(t)
		addr := sample.AccAddress()
		pair, err := common.NewTokenPairFromStr("LUNA:UST") // memeing
		require.NoError(t, err)

		dep.mockVpoolKeeper.
			EXPECT().
			GetSettlementPrice(gomock.Eq(ctx), gomock.Eq(pair)).
			Return(sdk.NewDec(1000), error(nil))

		// this means that the user
		// has bought 100 contracts
		// for an open notional of 1_000 coins
		// which means entry price is 10
		// now price is 1_000
		// which means current pos value is 100_000
		// now we need to return the user the profits
		// which are 99000 coins
		// we also need to return margin which is 100coin
		// so total is 99_100 coin
		pos := &types.Position{
			Address:      addr.String(),
			Pair:         pair.String(),
			Size_:        sdk.NewDec(100),
			Margin:       sdk.NewDec(100),
			OpenNotional: sdk.NewDec(1000),
		}
		err = k.Positions().Create(ctx, pos)
		require.NoError(t, err)

		coins, err := k.SettlePosition(ctx, pair, addr.String())
		require.NoError(t, err)
		require.Equal(t, coins, sdk.NewCoins(sdk.NewInt64Coin("todo", 99100))) // todo(mercilex): modify denom once transfer is impl
	})

	t.Run("position size is zero", func(t *testing.T) {
		k, _, ctx := getKeeper(t)
		addr := sample.AccAddress()
		pair, err := common.NewTokenPairFromStr("LUNA:UST") // memeing
		require.NoError(t, err)
		err = k.Positions().Create(ctx, &types.Position{
			Address: addr.String(),
			Pair:    pair.String(),
			Size_:   sdk.ZeroDec(),
		})
		require.NoError(t, err)

		coins, err := k.SettlePosition(ctx, pair, addr.String())
		require.ErrorIs(t, err, types.ErrPositionSizeZero)
		require.Len(t, coins, 0)
	})
}

// TODO(mercilex): copied from clearing_house_test.go, to cleanup

type mockedDependencies struct {
	mockAccountKeeper *mock.MockAccountKeeper
	mockBankKeeper    *mock.MockBankKeeper
	mockPriceKeeper   *mock.MockPriceKeeper
	mockVpoolKeeper   *mock.MockVpoolKeeper
}

func getKeeper(t *testing.T) (keeper.Keeper, mockedDependencies, sdk.Context) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.StoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	protoCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	params := initParamsKeeper(protoCodec, codec.NewLegacyAmino(), storeKey, memStoreKey)

	subSpace, found := params.GetSubspace(types.ModuleName)
	require.True(t, found)

	ctrl := gomock.NewController(t)
	mockedAccountKeeper := mock.NewMockAccountKeeper(ctrl)
	mockedBankKeeper := mock.NewMockBankKeeper(ctrl)
	mockedPriceKeeper := mock.NewMockPriceKeeper(ctrl)
	mockedVpoolKeeper := mock.NewMockVpoolKeeper(ctrl)

	mockedAccountKeeper.
		EXPECT().GetModuleAddress(types.ModuleName).
		Return(authtypes.NewModuleAddress(types.ModuleName))

	k := keeper.NewKeeper(
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
	paramsKeeper.Subspace(types.ModuleName)

	return paramsKeeper
}
