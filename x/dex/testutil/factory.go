package testutil

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/keeper"
	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

func CreateKeepers(t *testing.T, storeKey sdk.StoreKey) (keeper.Keeper, types.AccountKeeper, types.BankKeeper, sdk.Context, codec.ProtoCodec) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	accountKeeper := NewAccountKeeper(t, stateStore, cdc)
	bankKeeper := NewBankKeeper(t, stateStore, accountKeeper, cdc)
	dexKeeper := NewDexKeeper(t, storeKey, stateStore, accountKeeper, bankKeeper, cdc)

	return dexKeeper, accountKeeper, bankKeeper, ctx, *cdc
}
