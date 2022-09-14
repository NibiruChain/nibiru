package keys_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	db "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/collections"
	"github.com/NibiruChain/nibiru/collections/keys"
)

// deps is repeated but, don't want to create cross pkg dependencies
func deps() (sdk.StoreKey, sdk.Context, codec.BinaryCodec) {
	sk := sdk.NewKVStoreKey("mock")
	dbm := db.NewMemDB()
	ms := store.NewCommitMultiStore(dbm)
	ms.MountStoreWithDB(sk, types.StoreTypeIAVL, dbm)
	if err := ms.LoadLatestVersion(); err != nil {
		panic(err)
	}
	return sk, sdk.Context{}.WithMultiStore(ms).WithGasMeter(sdk.NewGasMeter(1_000_000_000)), codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}

func TestRangeBounds(t *testing.T) {
	sk, ctx, cdc := deps()

	ks := collections.NewKeySet[keys.Uint64Key](cdc, sk, 0)

	ks.Insert(ctx, 1)
	ks.Insert(ctx, 2)
	ks.Insert(ctx, 3)
	ks.Insert(ctx, 4)
	ks.Insert(ctx, 5)
	ks.Insert(ctx, 6)

	// let's range (1-5]; expected: 2..5
	start := keys.Exclusive[keys.Uint64Key](1)
	end := keys.Inclusive[keys.Uint64Key](5)
	rng := keys.NewRange[keys.Uint64Key]().Start(start).End(end)
	result := ks.Iterate(ctx, rng).Keys()
	require.Equal(t, []keys.Uint64Key{2, 3, 4, 5}, result)

	// let's range [1-5); expected 1..4
	start = keys.Inclusive[keys.Uint64Key](1)
	end = keys.Exclusive[keys.Uint64Key](5)
	rng = keys.NewRange[keys.Uint64Key]().Start(start).End(end)
	result = ks.Iterate(ctx, rng).Keys()
	require.Equal(t, []keys.Uint64Key{1, 2, 3, 4}, result)

	// let's range [1-5) descending; expected 4..1
	rng = keys.NewRange[keys.Uint64Key]().Start(start).End(end).Descending()
	result = ks.Iterate(ctx, rng).Keys()
	require.Equal(t, []keys.Uint64Key{4, 3, 2, 1}, result)
}
