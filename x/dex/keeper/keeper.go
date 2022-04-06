package keeper

import (
	"fmt"

	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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
) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
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
	store.Set(types.KeyNextGlobalPoolNumber,
		k.cdc.MustMarshal(&gogotypes.UInt64Value{Value: poolNumber}))
}

/*
Retrieves the next pool id number to use when creating a new pool.
This function is idempotent (does not change state).

args
  ctx: the cosmos-sdk context

ret
  uint64: a pool id number
*/
func (k Keeper) GetNextPoolNumber(ctx sdk.Context) (poolNumber uint64) {
	bz := ctx.KVStore(k.storeKey).Get(types.KeyNextGlobalPoolNumber)
	if bz == nil {
		panic(fmt.Errorf("pool number has not been initialized -- Should have been done in InitGenesis"))
	}
	val := gogotypes.UInt64Value{}
	k.cdc.MustUnmarshal(bz, &val)
	return val.GetValue()
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
Panics if the bytes could not be unmarshalled to a Pool proto object.

args
  ctx: the cosmos-sdk context
  poolId: the pool id number

ret
  pool: a Pool proto object
*/
func (k Keeper) FetchPool(ctx sdk.Context, poolId uint64) (pool types.Pool) {
	store := ctx.KVStore(k.storeKey)
	k.cdc.MustUnmarshal(store.Get(types.GetKeyPrefixPools(poolId)), &pool)
	return pool
}

/*
Writes a pool to the state.
Panics if the pool proto could not be marshalled.

args:
  - ctx: the cosmos-sdk context
  - pool: the Pool proto object
*/
func (k Keeper) SetPool(ctx sdk.Context, pool types.Pool) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetKeyPrefixPools(pool.Id), k.cdc.MustMarshal(&pool))
}

/*
Mints new pool share tokens and sends them to an account.

args:
  ctx: the cosmos-sdk context
  poolId: the pool id number
  recipientAddr: the address of the recipient
  amountPoolShares: the amount of pool shares to mint to the recipient

ret:
  err: returns an error if something errored out
*/
func (k Keeper) MintPoolShareToAccount(ctx sdk.Context, poolId uint64, recipientAddr sdk.AccAddress, amountPoolShares sdk.Int) (err error) {
	newCoins := sdk.Coins{
		sdk.NewCoin(types.GetPoolShareBaseDenom(poolId), amountPoolShares),
	}

	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, newCoins)
	if err != nil {
		return err
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, newCoins)
	if err != nil {
		return err
	}

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
  poolId: the pool id number
  err: an error if any occurred
*/
func (k Keeper) NewPool(
	ctx sdk.Context,
	sender sdk.AccAddress,
	poolParams types.PoolParams,
	poolAssets []types.PoolAsset,
) (poolId uint64, err error) {
	if len(poolAssets) < types.MinPoolAssets {
		return uint64(0), types.ErrTooFewPoolAssets
	}

	if len(poolAssets) > types.MaxPoolAssets {
		return uint64(0), types.ErrTooManyPoolAssets
	}

	poolId = k.GetNextPoolNumberAndIncrement(ctx)
	poolName := fmt.Sprintf("matrix-pool-%d", poolId)
	// Create a new account for the pool to hold funds.
	poolAccount := k.accountKeeper.NewAccount(ctx, authtypes.NewEmptyModuleAccount(poolName))
	k.accountKeeper.SetAccount(ctx, poolAccount)

	pool, err := types.NewPool(poolId, poolAccount.GetAddress(), poolParams, poolAssets)
	if err != nil {
		return uint64(0), err
	}

	// Transfer the PoolAssets tokens to the pool's module account from the user account.
	var coins sdk.Coins
	for _, asset := range poolAssets {
		coins = append(coins, asset.Token)
	}

	if err = k.bankKeeper.SendCoins(ctx, sender, poolAccount.GetAddress(), coins); err != nil {
		return 0, err
	}

	// Mint the initial 100.000000000000000000 pool share tokens to the sender
	if err = k.MintPoolShareToAccount(ctx, pool.Id, sender, types.InitPoolSharesSupply); err != nil {
		return 0, err
	}

	// Finally, add the share token's meta data to the bank keeper.
	poolShareBaseDenom := types.GetPoolShareBaseDenom(pool.Id)
	poolShareDisplayDenom := types.GetPoolShareDisplayDenom(pool.Id)
	k.bankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
		Description: fmt.Sprintf("The share token of the matrix dex pool %d", pool.Id),
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    poolShareBaseDenom,
				Exponent: 0,
			},
			{
				Denom:    poolShareDisplayDenom,
				Exponent: 18,
			},
		},
		Base:    poolShareBaseDenom,
		Display: poolShareDisplayDenom,
		Name:    fmt.Sprintf("Matrix Pool %d Share Token", pool.Id),
		Symbol:  poolShareDisplayDenom,
	})

	k.SetPool(ctx, pool)
	k.RecordTotalLiquidityIncrease(ctx, coins)

	return poolId, nil
}
