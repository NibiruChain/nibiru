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

// SetNextPoolNumber sets next pool number
func (k Keeper) SetNextPoolNumber(ctx sdk.Context, poolNumber uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&gogotypes.UInt64Value{Value: poolNumber})
	store.Set(types.KeyNextGlobalPoolNumber, bz)
}

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

// GetNextPoolNumberAndIncrement returns the next pool number, and increments the corresponding state entry
func (k Keeper) GetNextPoolNumberAndIncrement(ctx sdk.Context) uint64 {
	poolNumber := k.GetNextPoolNumber(ctx)
	k.SetNextPoolNumber(ctx, poolNumber+1)
	return poolNumber
}

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

func (k Keeper) NewPool(
	ctx sdk.Context,
	sender sdk.AccAddress,
	poolParams types.PoolParams,
	poolAssets []types.PoolAsset,
) (uint64, error) {
	poolId := k.GetNextPoolNumberAndIncrement(ctx)
	poolName := fmt.Sprintf("matrix-pool-%d", poolId)
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

	// TODO(heisenberg): implement
	// // Mint the initial 100.000000000000000000 share token to the sender
	// err = k.MintPoolShareToAccount(ctx, pool, sender, types.InitPoolSharesSupply)
	// if err != nil {
	// 	return 0, err
	// }

	// // Finally, add the share token's meta data to the bank keeper.
	// poolShareBaseDenom := types.GetPoolShareDenom(pool.GetId())
	// poolShareDisplayDenom := fmt.Sprintf("GAMM-%d", pool.GetId())
	// k.bankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
	// 	Description: fmt.Sprintf("The share token of the gamm pool %d", pool.GetId()),
	// 	DenomUnits: []*banktypes.DenomUnit{
	// 		{
	// 			Denom:    poolShareBaseDenom,
	// 			Exponent: 0,
	// 			Aliases: []string{
	// 				"attopoolshare",
	// 			},
	// 		},
	// 		{
	// 			Denom:    poolShareDisplayDenom,
	// 			Exponent: types.OneShareExponent,
	// 			Aliases:  nil,
	// 		},
	// 	},
	// 	Base:    poolShareBaseDenom,
	// 	Display: poolShareDisplayDenom,
	// })

	// err = k.SetPool(ctx, pool)
	// if err != nil {
	// 	return 0, err
	// }

	// k.hooks.AfterPoolCreated(ctx, sender, pool.GetId())
	// k.RecordTotalLiquidityIncrease(ctx, coins)

	return poolId, nil
}
