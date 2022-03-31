package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/MatrixDao/matrix/x/lockup/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// LockupKeeper provides a way to manage module storage.
type LockupKeeper struct {
	cdc      codec.Codec
	storeKey sdk.StoreKey

	ak types.AccountKeeper
	bk types.BankKeeper
	dk types.DistrKeeper
}

// NewKeeper returns an instance of Keeper.
func NewLockupKeeper(cdc codec.Codec, storeKey sdk.StoreKey, ak types.AccountKeeper,
	bk types.BankKeeper, dk types.DistrKeeper) LockupKeeper {
	return LockupKeeper{
		cdc:      cdc,
		storeKey: storeKey,
		ak:       ak,
		bk:       bk,
		dk:       dk,
	}
}

// Logger returns a logger instance.
func (k LockupKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
