package keeper

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/spot/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		paramstore paramtypes.Subspace

		accountKeeper types.AccountKeeper
		bankKeeper    types.BankKeeper
		distrKeeper   types.DistrKeeper
	}
)

/*
NewKeeper Creates a new keeper for the spot module.

args

	cdc: a codec
	storeKey: the key-value store key that this keeper uses
	ps: the param subspace for this keeper
	accountKeeper: the auth module\'s keeper for accounts
	bankKeeper: the bank module\'s keeper for bank transfers

ret

	Keeper: a keeper for the spot module
*/
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	ps paramtypes.Subspace,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distrKeeper types.DistrKeeper,
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
		distrKeeper:   distrKeeper,
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
GetNextPoolNumber Retrieves the next pool id number to use when creating a new pool.
This function is idempotent (does not change state).

args

	ctx: the cosmos-sdk context

ret

	uint64: a pool id number
*/
func (k Keeper) GetNextPoolNumber(ctx sdk.Context) (poolNumber uint64, err error) {
	bz := ctx.KVStore(k.storeKey).Get(types.KeyNextGlobalPoolNumber)
	if bz == nil {
		return poolNumber, fmt.Errorf("pool number has not been initialized -- Should have been done in InitGenesis")
	}
	val := gogotypes.UInt64Value{}
	k.cdc.MustUnmarshal(bz, &val)
	return val.GetValue(), err
}

/*
GetNextPoolNumberAndIncrement Returns the next pool id number, and increments the state's next pool id number by one
so that the next pool creation uses an autoincremented id number.

args

	ctx: the cosmos-sdk context

ret

	uint64: a pool id number
*/
func (k Keeper) GetNextPoolNumberAndIncrement(ctx sdk.Context) (uint64, error) {
	poolNumber, err := k.GetNextPoolNumber(ctx)
	if err != nil {
		return 0, err
	}
	k.SetNextPoolNumber(ctx, poolNumber+1)
	return poolNumber, err
}

/*
FetchPool Fetches a pool by id number.
Does not modify state.
Panics if the bytes could not be unmarshalled to a Pool proto object.

args

	ctx: the cosmos-sdk context
	poolId: the pool id number

ret

	pool: a Pool proto object
*/
func (k Keeper) FetchPool(ctx sdk.Context, poolId uint64) (pool types.Pool, err error) {
	store := ctx.KVStore(k.storeKey)
	k.cdc.MustUnmarshal(store.Get(types.GetKeyPrefixPools(poolId)), &pool)

	if len(pool.PoolAssets) == 0 {
		return pool, types.ErrPoolNotFound.Wrapf("could not find pool with id %d", poolId)
	}
	return pool, nil
}

/*
FetchPoolFromPair Given a pair of denom, find the corresponding pool id if it exists.

args:
  - denomA: One denom
  - denomB: A second denom

ret:
  - poolId: the pool id
  - err: error if any
*/
func (k Keeper) FetchPoolFromPair(ctx sdk.Context, denomA string, denomB string) (
	pool types.Pool, err error,
) {
	store := ctx.KVStore(k.storeKey)

	poolid := sdk.BigEndianToUint64(store.Get(types.GetDenomPrefixPoolIds(denomA, denomB)))
	pool, err = k.FetchPool(ctx, poolid)

	if err != nil {
		return pool, err
	}

	return pool, nil
}

/*
FetchAllPools fetch all pools from the store and returns them.
*/
func (k Keeper) FetchAllPools(ctx sdk.Context) (pools []types.Pool) {
	iterator := sdk.KVStorePrefixIterator(ctx.KVStore(k.storeKey), types.KeyPrefixPools)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var pool types.Pool
		k.cdc.MustUnmarshal(iterator.Value(), &pool)
		pools = append(pools, pool)
	}

	return pools
}

/*
SetPool Writes a pool to the state.
Panics if the pool proto could not be marshaled.

args:
  - ctx: the cosmos-sdk context
  - pool: the Pool proto object
*/
func (k Keeper) SetPool(ctx sdk.Context, pool types.Pool) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetKeyPrefixPools(pool.Id), k.cdc.MustMarshal(&pool))

	k.SetPoolIdByDenom(ctx, pool)
}

/*
SetPoolIdByDenom Writes a pool to the state accessible with the PoolId.
Panics if the pool proto could not be marshaled.

args:
  - ctx: the cosmos-sdk context
  - pool: the Pool proto object
*/
func (k Keeper) SetPoolIdByDenom(ctx sdk.Context, pool types.Pool) {
	denomA := pool.PoolAssets[0].Token.Denom
	denomB := pool.PoolAssets[1].Token.Denom

	store := ctx.KVStore(k.storeKey)
	store.Set(
		types.GetDenomPrefixPoolIds(denomA, denomB),
		sdk.Uint64ToBigEndian(pool.Id),
	)
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
func (k Keeper) mintPoolShareToAccount(ctx sdk.Context, poolId uint64, recipientAddr sdk.AccAddress, amountPoolShares sdk.Int) (err error) {
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
Burns takes an amount of pool shares from an account and burns them.
It's the inverse of mintPoolShareToAccount.

args:

	ctx: the cosmos-sdk context
	poolId: the pool id number
	recipientAddr: the address of the recipient
	amountPoolShares: the amount of pool shares to mint to the recipient

ret:

	err: returns an error if something errored out
*/
func (k Keeper) burnPoolShareFromAccount(
	ctx sdk.Context,
	fromAddr sdk.AccAddress,
	poolSharesOut sdk.Coin,
) (err error) {
	if err = k.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		fromAddr,
		types.ModuleName,
		sdk.Coins{poolSharesOut},
	); err != nil {
		return err
	}

	if err = k.bankKeeper.BurnCoins(
		ctx,
		types.ModuleName,
		sdk.Coins{poolSharesOut},
	); err != nil {
		return err
	}

	return nil
}

/*
NewPool Creates a brand new pool and writes it to the state.

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
		return 0, types.ErrTooFewPoolAssets
	}

	if len(poolAssets) > types.MaxPoolAssets {
		return 0, types.ErrTooManyPoolAssets
	}

	if !k.areAllAssetsWhitelisted(ctx, poolAssets) {
		return 0, types.ErrTokenNotAllowed
	}

	_, err = k.FetchPoolFromPair(ctx, poolAssets[0].Token.Denom, poolAssets[1].Token.Denom)
	if err == nil {
		return 0, types.ErrPoolWithSameAssetsExists
	}

	// send pool creation fee to community pool
	params := k.GetParams(ctx)
	err = k.distrKeeper.FundCommunityPool(ctx, params.PoolCreationFee, sender)
	if err != nil {
		return 0, err
	}

	poolId, err = k.GetNextPoolNumberAndIncrement(ctx)
	if err != nil {
		return 0, err
	}
	poolName := fmt.Sprintf("nibiru-pool-%d", poolId)
	// Create a new account for the pool to hold funds.
	poolAccount := k.accountKeeper.NewAccount(ctx, authtypes.NewEmptyModuleAccount(poolName))
	k.accountKeeper.SetAccount(ctx, poolAccount)

	pool, err := types.NewPool(poolId, poolAccount.GetAddress(), poolParams, poolAssets)
	if err != nil {
		return 0, err
	}

	// Transfer the PoolAssets tokens to the pool's module account from the user account.
	var coins sdk.Coins
	for _, asset := range poolAssets {
		coins = append(coins, asset.Token)
	}
	coins = sdk.NewCoins(coins...)

	if err = k.bankKeeper.SendCoins(ctx, sender, poolAccount.GetAddress(), coins); err != nil {
		return 0, err
	}

	// Mint the initial 100.000000000000000000 pool share tokens to the sender
	if err = k.mintPoolShareToAccount(ctx, pool.Id, sender, types.InitPoolSharesSupply); err != nil {
		return 0, err
	}

	// Finally, add the share token's meta data to the bank keeper.
	poolShareBaseDenom := types.GetPoolShareBaseDenom(pool.Id)
	poolShareDisplayDenom := types.GetPoolShareDisplayDenom(pool.Id)
	k.bankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
		Description: fmt.Sprintf("The share token of the nibiru spot amm pool %d", pool.Id),
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
		Name:    fmt.Sprintf("Nibiru Pool %d Share Token", pool.Id),
		Symbol:  poolShareDisplayDenom,
	})

	k.SetPool(ctx, pool)
	if err = k.RecordTotalLiquidityIncrease(ctx, coins); err != nil {
		return poolId, err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.EventPoolCreated{
		Creator: sender.String(),
		PoolId:  poolId,
	})
	if err != nil {
		return
	}

	return poolId, nil
}

// areAllAssetsWhitelisted checks if all assets are in whitelist
func (k Keeper) areAllAssetsWhitelisted(ctx sdk.Context, assets []types.PoolAsset) bool {
	whitelistedAssets := k.GetParams(ctx).GetWhitelistedAssetsAsMap()
	for _, a := range assets {
		if _, ok := whitelistedAssets[a.Token.Denom]; !ok {
			return false
		}
	}

	return true
}

/*
JoinPool Joins a pool without swapping leftover assets if the ratios don't exactly match the pool's asset ratios.

For example, if a pool has 100 pool shares, 100foo, 100bar,
and JoinPool is called with 75foo and bar, only 50foo and 50bar would be deposited.
25foo in remCoins would be returned to the user, along with 50 pool shares would be minted
and given to the user.

Inverse of ExitPool.

args:
  - ctx: the cosmos-sdk context
  - joinerAddr: the user who wishes to withdraw tokens
  - poolId: the pool's numeric id
  - tokensIn: the amount of liquidity to provide

ret:
  - pool: the updated pool after joining
  - numSharesOut: the pool shares minted and returned to the user
  - remCoins: the number of remaining coins from the user's initial deposit attempt
  - err: error if any
*/
func (k Keeper) JoinPool(
	ctx sdk.Context,
	joinerAddr sdk.AccAddress,
	poolId uint64,
	tokensIn sdk.Coins,
	shouldSwap bool,
) (pool types.Pool, numSharesOut sdk.Coin, remCoins sdk.Coins, err error) {
	pool, _ = k.FetchPool(ctx, poolId)

	if len(tokensIn) != len(pool.PoolAssets) && !shouldSwap {
		return pool, numSharesOut, remCoins, errors.New("too few assets to join this pool")
	}

	poolAddr := pool.GetAddress()

	var numShares sdk.Int
	if !shouldSwap || pool.PoolParams.PoolType == types.PoolType_STABLESWAP {
		numShares, remCoins, err = pool.AddTokensToPool(tokensIn)
	} else {
		numShares, remCoins, err = pool.AddAllTokensToPool(tokensIn)
	}
	if err != nil {
		return types.Pool{}, sdk.Coin{}, sdk.Coins{}, err
	}

	tokensConsumed := tokensIn.Sub(remCoins)

	// take coins from joiner to pool
	if err = k.bankKeeper.SendCoins(
		ctx,
		/*from=*/ joinerAddr,
		/*to=*/ poolAddr,
		/*amount=*/ tokensConsumed,
	); err != nil {
		return pool, numSharesOut, remCoins, err
	}

	// give joiner LP shares
	if err = k.mintPoolShareToAccount(
		ctx,
		/*from=*/ pool.Id,
		/*to=*/ joinerAddr,
		/*amount=*/ numShares,
	); err != nil {
		return types.Pool{}, sdk.Coin{}, sdk.Coins{}, err
	}

	// record changes to store
	k.SetPool(ctx, pool)
	if err = k.RecordTotalLiquidityIncrease(ctx, tokensConsumed); err != nil {
		return pool, numSharesOut, remCoins, err
	}

	poolSharesOut := sdk.NewCoin(pool.TotalShares.Denom, numShares)

	err = ctx.EventManager().EmitTypedEvent(&types.EventPoolJoined{
		Address:       joinerAddr.String(),
		PoolId:        poolId,
		TokensIn:      tokensIn,
		PoolSharesOut: poolSharesOut,
		RemCoins:      remCoins,
	})
	if err != nil {
		return
	}

	return pool, poolSharesOut, remCoins, nil
}

/*
ExitPool Exits a pool by taking out tokens relative to the amount of pool shares
in proportion to the total amount of pool shares.

For example, if a pool has 100 pool shares and ExitPool is called with 50 pool shares,
half of the tokens (minus exit fees) are returned to the user.

Inverse of JoinPool.

Throws an error if the provided pool shares doesn't match up with the pool's actual pool share.

args:
  - ctx: the cosmos-sdk context
  - sender: the user who wishes to withdraw tokens
  - poolId: the pool's numeric id
  - poolSharesOut: the amount of pool shares to burn

ret:
  - tokensOut: the amount of liquidity withdrawn from the pool
  - err: error if any
*/
func (k Keeper) ExitPool(
	ctx sdk.Context,
	sender sdk.AccAddress,
	poolId uint64,
	poolSharesOut sdk.Coin,
) (tokensOut sdk.Coins, err error) {
	pool, _ := k.FetchPool(ctx, poolId)

	// sanity checks
	if poolSharesOut.Denom != pool.TotalShares.Denom {
		return sdk.Coins{},
			fmt.Errorf("invalid pool share denom. expected %s, got %s",
				pool.TotalShares.Denom,
				poolSharesOut.Denom,
			)
	}

	if poolSharesOut.Amount.GT(pool.TotalShares.Amount) ||
		poolSharesOut.Amount.LTE(sdk.ZeroInt()) {
		return sdk.Coins{}, fmt.Errorf(
			"invalid number of pool shares %s must be between 0 and %s",
			poolSharesOut.Amount, pool.TotalShares.Amount,
		)
	}

	// calculate withdrawn liquidity
	tokensOut, fees, err := pool.ExitPool(poolSharesOut.Amount)
	if err != nil {
		return sdk.Coins{}, err
	}

	// apply exchange of pool shares for tokens
	if err = k.bankKeeper.SendCoins(ctx, pool.GetAddress(), sender, tokensOut); err != nil {
		return sdk.Coins{}, err
	}

	if err = k.burnPoolShareFromAccount(ctx, sender, poolSharesOut); err != nil {
		return sdk.Coins{}, err
	}

	// record state changes
	k.SetPool(ctx, pool)
	if err = k.RecordTotalLiquidityDecrease(ctx, tokensOut); err != nil {
		return sdk.Coins{}, err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.EventPoolExited{
		Address:      sender.String(),
		PoolId:       poolId,
		PoolSharesIn: poolSharesOut,
		TokensOut:    tokensOut,
		Fees:         fees,
	})
	if err != nil {
		return sdk.Coins{}, err
	}

	return tokensOut, nil
}
