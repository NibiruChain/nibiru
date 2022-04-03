package types

import (
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
