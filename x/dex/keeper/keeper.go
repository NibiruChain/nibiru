package keeper

import (
	"fmt"

	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/tendermint/tendermint/libs/log"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		paramstore paramtypes.Subspace

		accountKeeper types.AccountKeeper
		bankKeeper    types.BankKeeper
	}
)

/*
Creates a new keeper for the dex module.

args
  cdc: a codec
  storeKey: the key-value store key that this keeper uses
  ps: the param subspace for this keeper
  accountKeeper: the auth module\'s keeper for accounts
  bankKeeper: the bank module\'s keeper for bank transfers

ret
  Keeper: a keeper for the dex module
*/
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	ps paramtypes.Subspace,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		paramstore:    ps,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

/*
Sets the next pool id that should be chosen when a new pool is created.
This function changes the state.

args
  ctx: the cosmos-sdk context
  poolNumber: the numeric id of the next pool number to use
*/
func (k Keeper) SetNextPoolNumber(ctx sdk.Context, poolNumber uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&gogotypes.UInt64Value{Value: poolNumber})
	store.Set(types.KeyNextGlobalPoolNumber, bz)
}

/*
Retrieves the next pool id number to use when creating a new pool.
This function is idempotent (does not change state).

args
  ctx: the cosmos-sdk context

ret
  uint64: a pool id number
*/
func (k Keeper) GetNextPoolNumber(ctx sdk.Context) uint64 {
	var poolNumber uint64
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.KeyNextGlobalPoolNumber)
	if bz == nil {
		panic(fmt.Errorf("pool number has not been initialized -- Should have been done in InitGenesis"))
	} else {
		val := gogotypes.UInt64Value{}

		err := k.cdc.Unmarshal(bz, &val)
		if err != nil {
			panic(err)
		}

		poolNumber = val.GetValue()
	}

	return poolNumber
}

/*
Returns the next pool id number, and increments the state's next pool id number by one
so that the next pool creation uses an autoincremented id number.

args
  ctx: the cosmos-sdk context

ret
  uint64: a pool id number
*/
func (k Keeper) GetNextPoolNumberAndIncrement(ctx sdk.Context) uint64 {
	poolNumber := k.GetNextPoolNumber(ctx)
	k.SetNextPoolNumber(ctx, poolNumber+1)
	return poolNumber
}

/*
Fetches a pool by id number.
Does not modify state.

args
  ctx: the cosmos-sdk context
  poolId: the pool id number

ret
  Pool: a Pool proto object
  error: an error if any occurred
*/
func (k Keeper) FetchPool(ctx sdk.Context, poolId uint64) (types.Pool, error) {
	store := ctx.KVStore(k.storeKey)
	poolKey := types.GetKeyPrefixPools(poolId)
	bz := store.Get(poolKey)

	var pool types.Pool
	err := pool.Unmarshal(bz)
	if err != nil {
		return types.Pool{}, err
	}

	return pool, nil
}

/*
Writes a pool to the state.

args
  ctx: the cosmos-sdk context
  Pool: a Pool proto object

ret
  error: an error if any occurred
*/
func (k Keeper) SetPool(ctx sdk.Context, pool types.Pool) error {
	bz, err := pool.Marshal()
	if err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	poolKey := types.GetKeyPrefixPools(pool.Id)
	store.Set(poolKey, bz)

	return nil
}

/*
Creates a brand new pool and writes it to the state.

args
  ctx: the cosmos-sdk context
  sender: the pool creator's address
  poolParams: parameters of the pool, represented by a PoolParams proto object
  poolAssets: initial assets in the pool, represented by a PoolAssets proto object array

ret
  uint64: the pool id number
  error: an error if any occurred
*/
func (k Keeper) NewPool(
	ctx sdk.Context,
	sender sdk.AccAddress,
	poolParams types.PoolParams,
	poolAssets []types.PoolAsset,
) (uint64, error) {
	poolId := k.GetNextPoolNumberAndIncrement(ctx)
	poolName := fmt.Sprintf("matrix-pool-%d", poolId)
	// Create a new account for the pool to hold funds.
	poolAccount := k.accountKeeper.NewAccount(ctx, authtypes.NewEmptyModuleAccount(poolName))

	k.accountKeeper.SetAccount(ctx, poolAccount)

	pool := types.Pool{
		Id:         poolId,
		Address:    poolAccount.GetAddress().String(),
		PoolParams: poolParams,
		PoolAssets: poolAssets,
	}

	err := k.SetPool(ctx, pool)
	if err != nil {
		return 0, err
	}

	// Transfer the PoolAssets tokens to the pool's module account from the user account.
	var coins sdk.Coins
	for _, asset := range poolAssets {
		coins = append(coins, asset.Token)
	}
	coins = coins.Sort()

	err = k.bankKeeper.SendCoins(ctx, sender, poolAccount.GetAddress(), coins)
	if err != nil {
		return 0, err
	}

	// TODO(heisenberg): finish implementation of setting up the pool

	return poolId, nil
}
