package types

import (
	"errors"
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validates a PoolAsset amount and weights.
func (poolAsset PoolAsset) Validate() error {
	if poolAsset.Token.Amount.LTE(sdk.ZeroInt()) {
		return fmt.Errorf("can't add the zero or negative balance of token")
	}

	if poolAsset.Weight.LTE(sdk.ZeroInt()) {
		return fmt.Errorf("a token's weight in the pool must be greater than 0")
	}

	if poolAsset.Weight.GTE(MaxUserSpecifiedWeight.MulRaw(GuaranteedWeightPrecision)) {
		return fmt.Errorf("a token's weight in the pool must be less than 1^50")
	}

	return nil
}

/*
Returns all of the coins contained in the pool's assets.

ret:
  - coins: the coin denoms and amounts that the pool contains, aka the pool total liquidity
*/
func GetPoolLiquidity(poolAssets []PoolAsset) (coins sdk.Coins) {
	coins = sdk.Coins{}
	for _, asset := range poolAssets {
		coins = coins.Add(asset.Token)
	}
	return coins
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
