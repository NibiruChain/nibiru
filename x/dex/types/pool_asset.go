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
