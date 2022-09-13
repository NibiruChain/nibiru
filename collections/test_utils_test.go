package collections

import (
	"encoding/json"
	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	db "github.com/tendermint/tm-db"
)

var _ Object = (*lock)(nil)

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

type lock struct {
	Start   keys.Uint8Key
	End     keys.Uint8Key
	ID      keys.Uint8Key
	Address keys.StringKey
}

func (l lock) Reset() {
	//TODO implement me
	panic("implement me")
}

func (l lock) String() string {
	//TODO implement me
	panic("implement me")
}

func (l lock) ProtoMessage() {
	//TODO implement me
	panic("implement me")
}

func (l *lock) Marshal() ([]byte, error) {
	return json.Marshal(l)
}

func (l *lock) MarshalTo(data []byte) (n int, err error) {
	panic("implement me")
}

func (l *lock) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (l lock) Size() int {
	panic("implement me")
}

func (l lock) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &l)
}
