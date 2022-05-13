package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/x/vpool/types"
)

const NUSDPair = "BTC:NUSD"

func VpoolKeeper(t *testing.T, pricefeedKeeper types.PricefeedKeeper) (
	vpoolKeeper Keeper, ctx sdk.Context,
) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	vpoolKeeper = NewKeeper(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		storeKey,
		pricefeedKeeper,
	)
	ctx = sdk.NewContext(stateStore, tmproto.Header{}, false, nil)

	return vpoolKeeper, ctx
}

func getSamplePool() *types.Pool {
	ratioLimit, _ := sdk.NewDecFromStr("0.9")
	fluctuationLimit, _ := sdk.NewDecFromStr("0.1")

	pool := types.NewPool(
		NUSDPair,
		ratioLimit,
		sdk.NewDec(10_000_000),
		sdk.NewDec(5_000_000),
		fluctuationLimit,
	)

	return pool
}
