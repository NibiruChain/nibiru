package collections

import (
	db "github.com/cometbft/cometbft-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func deps() (types.StoreKey, sdk.Context, codec.BinaryCodec) {
	sk := sdk.NewKVStoreKey("mock")
	dbm := db.NewMemDB()
	ms := store.NewCommitMultiStore(dbm)
	ms.MountStoreWithDB(sk, types.StoreTypeIAVL, dbm)
	if err := ms.LoadLatestVersion(); err != nil {
		panic(err)
	}

	return sk,
		sdk.Context{}.
			WithMultiStore(ms).
			WithGasMeter(sdk.NewGasMeter(1_000_000_000)),
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}
