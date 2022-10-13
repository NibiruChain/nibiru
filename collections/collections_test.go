package collections

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	db "github.com/tendermint/tm-db"
)

func deps() (sdk.StoreKey, sdk.Context, codec.BinaryCodec) {
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

// stringValue is a ValueEncoder for string, used for testing.
type stringValue struct{}

func (s stringValue) ValueEncode(value string) []byte { return []byte(value) }
func (s stringValue) ValueDecode(b []byte) string     { return string(b) }
func (s stringValue) Stringify(value string) string   { return value }
func (s stringValue) Name() string                    { return "test string" }

// jsonValue is a ValueEncoder for objects to be turned into json.
// used for testing.
type jsonValue[T any] struct{}

func (jsonValue[T]) ValueEncode(value T) []byte {
	b, _ := json.Marshal(value)
	return b
}

func (jsonValue[T]) ValueDecode(b []byte) T {
	v := new(T)
	_ = json.Unmarshal(b, v)
	return *v
}

func (jsonValue[T]) Stringify(v T) string { return fmt.Sprintf("%#v", v) }
func (jsonValue[T]) Name() string {
	var t T
	return fmt.Sprintf("json-value-%T", t)
}
