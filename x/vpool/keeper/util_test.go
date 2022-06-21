package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/x/vpool/types"
)

var BTCNusdPair = common.AssetPair{
	Token0: "BTC",
	Token1: "NUSD",
}

var ETHNusdPair = common.AssetPair{
	Token0: "ETH",
	Token1: "NUSD",
}

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
		storeKey, pricefeedKeeper,
	)
	ctx = sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	return vpoolKeeper, ctx
}

func getSamplePool() *types.Pool {
	ratioLimit, _ := sdk.NewDecFromStr("0.9")
	fluctuationLimit, _ := sdk.NewDecFromStr("0.1")
	maxOracleSpreadRatio := sdk.MustNewDecFromStr("0.1")

	pool := types.NewPool(
		BTCNusdPair,
		ratioLimit,
		sdk.NewDec(10_000_000),
		sdk.NewDec(5_000_000),
		fluctuationLimit,
		maxOracleSpreadRatio,
	)

	return pool
}
