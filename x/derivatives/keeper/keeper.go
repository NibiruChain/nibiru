package keeper

import (
	"fmt"

	derivativesv1 "github.com/MatrixDao/matrix/api/derivatives"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/MatrixDao/matrix/x/derivatives/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type AMM interface {
	PoolExists(ctx sdk.Context, poolName string) bool
	DirectSwap(ctx sdk.Context, poolName string, asset string, amt sdk.Int) (swappedAmount sdk.Int, err error)
	InverseSwap(ctx sdk.Context, poolName string, asset string, amt sdk.Int) (swappedAmount sdk.Int, err error)
}

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	memKey   storetypes.StoreKey

	store derivativesv1.StateStore

	// imports
	bk bank.Keeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		memKey:   memKey,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k *Keeper) Stopped(ctx sdk.Context) bool {
	params, err := k.store.ParamsTable().Get(ctx)
	if err != nil {
		panic(err)
	}

	return params.Stopped
}
