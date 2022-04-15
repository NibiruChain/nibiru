package types

import (
	"errors"
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
Returns the *base* denomination of a pool share token for a given poolId.

args:
  poolId: the pool id number

ret:
  poolDenom: the pool denomination name of the poolId
*/
func GetPoolShareBaseDenom(poolId uint64) (poolDenom string) {
	return fmt.Sprintf("matrix/pool/%d", poolId)
}

/*
Returns the *display* denomination of a pool share token for a given poolId.
Display denom means the denomination showed to the user, which could be many exponents greater than the base denom.
e.g. 1 atom is the display denom, but 10^6 uatom is the base denom.

In Matrix, a display denom is 10^18 base denoms.

args:
  poolId: the pool id number

ret:
  poolDenom: the pool denomination name of the poolId
*/
func GetPoolShareDisplayDenom(poolId uint64) (poolDenom string) {
	return fmt.Sprintf("MATRIX-POOL-%d", poolId)
}

/*
Creates a new pool and sets the initial assets.

args:
  poolId: the pool numeric id
  poolAccountAddr: the pool's account address for holding funds
  poolParams: pool configuration options
  poolAssets: the initial pool assets and weights

ret:
  pool: a new pool
  err: error if any
*/
func NewPool(
	poolId uint64,
	poolAccountAddr sdk.Address,
	poolParams PoolParams,
	poolAssets []PoolAsset,
) (pool Pool, err error) {
	pool = Pool{
		Id:          poolId,
		Address:     poolAccountAddr.String(),
		PoolParams:  poolParams,
		PoolAssets:  nil,
		TotalWeight: sdk.ZeroInt(),
		TotalShares: sdk.NewCoin(GetPoolShareBaseDenom(poolId), InitPoolSharesSupply),
	}

	err = pool.setInitialPoolAssets(poolAssets)
	if err != nil {
		return Pool{}, err
	}

	return pool, nil
}

/*
Adds tokens to a pool and updates the pool balances (i.e. liquidity).

args:
  - tokensIn: the tokens to add to the pool

ret:
  - numShares: the number of LP shares given to the user for the deposit
  - remCoins: the number of coins remaining after the deposit
  - err: error if any
*/
func (pool *Pool) JoinPool(tokensIn sdk.Coins) (numShares sdk.Int, remCoins sdk.Coins, err error) {
	if tokensIn.Len() != len(pool.PoolAssets) {
		return sdk.ZeroInt(), sdk.NewCoins(), errors.New("wrong number of assets to deposit into the pool")
	}

	// Add all exact coins we can (no swap)
	numShares, remCoins, err = pool.maximalSharesFromExactRatioJoin(tokensIn)
	if err != nil {
		return sdk.ZeroInt(), sdk.NewCoins(), err
	}

	if err := pool.updateLiquidity(numShares, tokensIn.Sub(remCoins)); err != nil {
		return sdk.ZeroInt(), sdk.NewCoins(), err
	}

	return numShares, remCoins, nil
}

/*
Fetch the pool's address as an sdk.Address.
*/
func (pool Pool) GetAddress() (addr sdk.AccAddress) {
	addr, err := sdk.AccAddressFromBech32(pool.Address)
	if err != nil {
		panic(fmt.Sprintf("could not bech32 decode address of pool with id: %d", pool.Id))
	}
	return addr
}

/*
Given the amount of pool shares to exit, calculates the amount of coins to exit
from the pool and modifies the pool. Accounts for an exit fee, if any, on the pool.

args:
  - exitingShares: the number of pool shares to exit from the pool
*/
func (pool *Pool) ExitPool(exitingShares sdk.Int) (
	exitedCoins sdk.Coins, err error,
) {
	if exitingShares.GT(pool.TotalShares.Amount) {
		return sdk.Coins{}, errors.New("too many shares out")
	}

	exitedCoins, err = pool.tokensOutFromExactShares(exitingShares)
	if err != nil {
		return sdk.Coins{}, err
	}

	// update the pool's balances
	for _, exitedCoin := range exitedCoins {
		err = pool.SubtractPoolAssetBalance(exitedCoin.Denom, exitedCoin.Amount)
		if err != nil {
			return sdk.Coins{}, err
		}
	}

	pool.TotalShares = sdk.NewCoin(pool.TotalShares.Denom, pool.TotalShares.Amount.Sub(exitingShares))
	return exitedCoins, nil
}
