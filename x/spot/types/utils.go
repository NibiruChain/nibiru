package types

import (
	fmt "fmt"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
given a pool, find the poolAsset with a given denom.

args:
  - poolAssets: all of the pool's assets. Must be sorted.
  - denom: the denom string to search for

ret:
  - index: the index of the denom in the pool assets
  - poolAsset: the pool asset itself
  - err: error if any
*/
func (pool Pool) getPoolAssetAndIndex(denom string) (
	index int, poolAsset PoolAsset, err error,
) {
	if denom == "" {
		return -1, PoolAsset{}, fmt.Errorf("empty denom")
	}

	if len(pool.PoolAssets) == 0 {
		return -1, PoolAsset{}, fmt.Errorf("empty pool assets")
	}

	// binary search for the asset. poolAssets must be sorted.
	i := sort.Search(len(pool.PoolAssets), func(i int) bool {
		compare := strings.Compare(pool.PoolAssets[i].Token.Denom, denom)
		return compare >= 0
	})

	if i < 0 || i >= len(pool.PoolAssets) || pool.PoolAssets[i].Token.Denom != denom {
		return -1, PoolAsset{}, ErrTokenDenomNotFound.Wrapf("could not find denom %s in pool id %d", denom, pool.Id)
	}

	return i, pool.PoolAssets[i], nil
}

/*
Maps poolAssets to its underlying coins.

ret:
  - coins: all the coins in the pool assets

args:
  - poolAssets: the slice of pool assets
*/
func (pool Pool) PoolBalances() sdk.Coins {
	coins := sdk.NewCoins()
	for _, asset := range pool.PoolAssets {
		coins = coins.Add(asset.Token)
	}
	return coins
}

/*
Sorts poolAssets in place by denom, lexicographically increasing.

args:
  - poolAssets: the pool assets to sort
*/
func sortPoolAssetsByDenom(poolAssets []PoolAsset) {
	sort.Slice(poolAssets, func(i, j int) bool {
		return strings.Compare(poolAssets[i].Token.Denom, poolAssets[j].Token.Denom) == -1
	})
}
