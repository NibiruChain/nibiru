package examples

import (
	"github.com/NibiruChain/nibiru/v2/x/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	crypto "github.com/cosmos/cosmos-sdk/crypto/types"
	storagetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type AccountKeeper struct {
	AccountNumber collections.Sequence
	Accounts      collections.Map[sdk.AccAddress, types.BaseAccount]
	Params        collections.Item[types.Params]
}

func NewAccountKeeper(sk storagetypes.StoreKey, cdc codec.BinaryCodec) *AccountKeeper {
	return &AccountKeeper{
		AccountNumber: collections.NewSequence(sk, 0),                                                                                     // namespace is unique across the module's collections types
		Accounts:      collections.NewMap(sk, 1, collections.AccAddressKeyEncoder, collections.ProtoValueEncoder[types.BaseAccount](cdc)), // we pass it the AccAddress key encoder and the base account value encoder.
		Params:        collections.NewItem(sk, 2, collections.ProtoValueEncoder[types.Params](cdc)),
	}
}

func (k AccountKeeper) CreateAccount(ctx sdk.Context, pubKey crypto.PubKey) {
	n := k.AccountNumber.Next(ctx)
	addr := sdk.AccAddress(pubKey.String())
	acc := types.BaseAccount{
		Address:       addr.String(),
		AccountNumber: n,
		Sequence:      0,
	}

	k.Accounts.Insert(ctx, addr, acc)
}

func (k AccountKeeper) GetAccount(ctx sdk.Context, addr sdk.AccAddress) (types.BaseAccount, error) {
	return k.Accounts.Get(ctx, addr)
}

func (k AccountKeeper) AllAccounts(ctx sdk.Context) []types.BaseAccount {
	return k.Accounts.Iterate(ctx, collections.Range[sdk.AccAddress]{}).Values()
}

func (k AccountKeeper) AllAddresses(ctx sdk.Context) []sdk.AccAddress {
	return k.Accounts.Iterate(ctx, collections.Range[sdk.AccAddress]{}).Keys()
}

func (k AccountKeeper) AccountsBetween(ctx sdk.Context, start, end sdk.AccAddress) []types.BaseAccount {
	rng := collections.Range[sdk.AccAddress]{}.
		StartInclusive(start).
		EndInclusive(end)
	// .Descending() for reverse order
	return k.Accounts.Iterate(ctx, rng).Values()
}
