package keeper

import (
	"github.com/MatrixDao/matrix/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
Returns all the coins corresponding to a slice of pool assets.

args:
  assets: a slice of pool assets

ret:
  coins: the coins from the pool assets
*/
func PoolAssetsCoins(assets []types.PoolAsset) (coins sdk.Coins) {
	coins = sdk.Coins{}
	for _, asset := range assets {
		coins = coins.Add(asset.Token)
	}
	return coins
}
