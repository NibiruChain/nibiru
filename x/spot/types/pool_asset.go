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
Subtracts an amount of coins from a pool's assets.
Throws an error if the final amount is less than zero.
*/
func (pool *Pool) SubtractPoolAssetBalance(assetDenom string, subAmt sdk.Int) (err error) {
	if subAmt.LT(sdk.ZeroInt()) {
		return errors.New("can't subtract a negative amount")
	}

	index, poolAsset, err := pool.getPoolAssetAndIndex(assetDenom)
	if err != nil {
		return err
	}

	// Update the supply of the asset
	poolAsset.Token.Amount = poolAsset.Token.Amount.Sub(subAmt)
	if poolAsset.Token.Amount.LT(sdk.ZeroInt()) {
		return errors.New("can't set the pool's balance of a token to be zero or negative")
	}
	pool.PoolAssets[index] = poolAsset
	return nil
}
