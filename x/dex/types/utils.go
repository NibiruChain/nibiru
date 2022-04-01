package types

import (
	fmt "fmt"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
Given a pool, find the poolAsset with a given denom.

args:
  - poolAssets: all of the pool's assets. Must be sorted.
  - denom: the denom string to search for

ret:
  - index: the index of the denom in the pool assets
  - poolAsset: the pool asset itself
  - err: error if any
*/
func getPoolAssetAndIndex(poolAssets []PoolAsset, denom string) (index int, poolAsset PoolAsset, err error) {
	if denom == "" {
		return -1, PoolAsset{}, fmt.Errorf("Empty denom.")
	}

	if len(poolAssets) == 0 {
		return -1, PoolAsset{}, fmt.Errorf("Empty pool assets.")
	}

	// binary search for the asset. poolAssets must be sorted.
	i := sort.Search(len(poolAssets), func(i int) bool {
		compare := strings.Compare(poolAssets[i].Token.Denom, denom)
		return compare == 0
	})

	if i < 0 || i >= len(poolAssets) {
		return -1, PoolAsset{}, fmt.Errorf("Did not find the PoolAsset (%s)", denom)
	}

	return i, poolAssets[i], nil
}

/*
Maps poolAssets to its underlying coins.

ret:
  - coins: all the coins in the pool assets

args:
  - poolAssets: the slice of pool assets
*/
func poolAssetsCoins(poolAssets []PoolAsset) (coins sdk.Coins) {
	coins = sdk.Coins{}
	for _, asset := range poolAssets {
		coins = coins.Add(asset.Token)
	}
	return coins
}
