package keeper

import (
	"github.com/NibiruChain/nibiru/x/vpool/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) Whitelist(ctx sdk.Context) Whitelist {
	return Whitelist{
		store: prefix.NewStore(
			ctx.KVStore(k.storeKey),
			types.WhitelistPrefix,
		),
	}
}

type Whitelist struct {
	store sdk.KVStore
}

// Add adds an address to the whitelist.
func (w Whitelist) Add(addr sdk.AccAddress) error {
	if w.store.Has(addr) {
		return types.ErrAlreadyInWhitelist.Wrapf("%s", addr.String())
	}

	w.store.Set(addr, []byte{})
	return nil
}

// Remove removes the provided address from the whitelist.
func (w Whitelist) Remove(addr sdk.AccAddress) error {
	if !w.store.Has(addr) {
		return types.ErrNotInWhitelist.Wrapf("%s", addr.String())
	}

	w.store.Delete(addr)
	return nil
}

// Iterate iterates over all the whitelisted addresses until it finishes
// or true is returned from the do function.
func (w Whitelist) Iterate(do func(addr sdk.AccAddress) (stop bool)) {
	iter := w.store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		addr := iter.Key()
		if do(addr) {
			break
		}
	}
}

func (w Whitelist) IsWhitelisted(addr sdk.AccAddress) bool {
	return w.store.Has(addr)
}
