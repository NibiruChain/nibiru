package types

import (
	"errors"
	"fmt"

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
	return fmt.Sprintf("nibiru/pool/%d", poolId)
}

/*
Returns the *display* denomination of a pool share token for a given poolId.
Display denom means the denomination showed to the user, which could be many exponents greater than the base denom.
e.g. 1 atom is the display denom, but 10^6 uatom is the base denom.

In Nibiru, a display denom is 10^18 base denoms.

args:

	poolId: the pool id number

ret:

	poolDenom: the pool denomination name of the poolId
*/
func GetPoolShareDisplayDenom(poolId uint64) (poolDenom string) {
	return fmt.Sprintf("NIBIRU-POOL-%d", poolId)
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
func (pool *Pool) AddTokensToPool(tokensIn sdk.Coins) (
	numShares sdk.Int, remCoins sdk.Coins, err error,
) {
	if tokensIn.Len() != len(pool.PoolAssets) {
		return sdk.ZeroInt(), sdk.Coins{}, errors.New("wrong number of assets to deposit into the pool")
	}

	// Calculate max amount of tokensIn we can deposit into pool (no swap)
	numShares, remCoins, err = pool.numSharesOutFromTokensIn(tokensIn)
	if err != nil {
		return sdk.ZeroInt(), sdk.Coins{}, err
	}

	if err := pool.incrementBalances(numShares, tokensIn.Sub(remCoins)); err != nil {
		return sdk.ZeroInt(), sdk.Coins{}, err
	}

	return numShares, remCoins, nil
}

/*
Adds tokens to a pool optimizing the amount of shares (swap + join) and updates the pool balances (i.e. liquidity).
We compute the swap and then join the pool.

args:
  - tokensIn: the tokens to add to the pool

ret:
  - numShares: the number of LP shares given to the user for the deposit
  - remCoins: the number of coins remaining after the deposit
  - err: error if any
*/
func (pool *Pool) AddAllTokensToPool(tokensIn sdk.Coins) (
	numShares sdk.Int, remCoins sdk.Coins, err error,
) {
	swapToken, err := pool.SwapForSwapAndJoin(tokensIn)
	if err != nil {
		return
	}
	if swapToken.Amount.LT(sdk.OneInt()) {
		return pool.AddTokensToPool(tokensIn)
	}

	index, _, err := pool.getPoolAssetAndIndex(swapToken.Denom)

	if err != nil {
		return
	}

	otherDenom := pool.PoolAssets[1-index].Token.Denom
	tokenOut, err := pool.CalcOutAmtGivenIn(
		/*tokenIn=*/ swapToken,
		/*tokenOutDenom=*/ otherDenom,
		/*noFee=*/ true,
	)

	if err != nil {
		return
	}

	err = pool.ApplySwap(swapToken, tokenOut)

	if err != nil {
		return
	}

	tokensIn = sdk.Coins{
		{
			Denom:  swapToken.Denom,
			Amount: tokensIn.AmountOfNoDenomValidation(swapToken.Denom).Sub(swapToken.Amount),
		},
		{
			Denom:  otherDenom,
			Amount: tokensIn.AmountOfNoDenomValidation(otherDenom).Add(tokenOut.Amount),
		},
	}.Sort()
	return pool.AddTokensToPool(tokensIn)
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

	exitedCoins, err = pool.TokensOutFromPoolSharesIn(exitingShares)
	if err != nil {
		return sdk.Coins{}, err
	}

	if !exitedCoins.IsValid() {
		return sdk.Coins{}, errors.New("not enough pool shares to withdraw")
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

/*
Updates the pool's asset liquidity using the provided tokens.

args:
  - tokens: the new token liquidity in the pool

ret:
  - err: error if any
*/
func (pool *Pool) updatePoolAssetBalances(tokens sdk.Coins) (err error) {
	// Ensures that there are no duplicate denoms, all denom's are valid,
	// and amount is > 0
	if len(tokens) != len(pool.PoolAssets) {
		return errors.New("provided tokens do not match number of assets in pool")
	}
	if err = tokens.Validate(); err != nil {
		return fmt.Errorf("provided coins are invalid, %v", err)
	}

	for _, coin := range tokens {
		assetIndex, existingAsset, err := pool.getPoolAssetAndIndex(coin.Denom)
		if err != nil {
			return err
		}
		existingAsset.Token = coin
		pool.PoolAssets[assetIndex].Token = coin
	}

	return nil
}

// setInitialPoolAssets sets the PoolAssets in the pool.
// It is only designed to be called at the pool's creation.
// If the same denom's PoolAsset exists, will return error.
// The list of PoolAssets must be sorted. This is done to enable fast searching for a PoolAsset by denomination.
func (p *Pool) setInitialPoolAssets(poolAssets []PoolAsset) (err error) {
	exists := make(map[string]bool)

	newTotalWeight := sdk.ZeroInt()
	scaledPoolAssets := make([]PoolAsset, 0, len(poolAssets))

	for _, asset := range poolAssets {
		if err = asset.Validate(); err != nil {
			return err
		}

		if exists[asset.Token.Denom] {
			return fmt.Errorf("same PoolAsset already exists")
		}
		exists[asset.Token.Denom] = true

		// Scale weight from the user provided weight to the correct internal weight
		asset.Weight = asset.Weight.MulRaw(GuaranteedWeightPrecision)
		scaledPoolAssets = append(scaledPoolAssets, asset)
		newTotalWeight = newTotalWeight.Add(asset.Weight)
	}

	p.PoolAssets = scaledPoolAssets
	sortPoolAssetsByDenom(p.PoolAssets)

	p.TotalWeight = newTotalWeight

	return nil
}
