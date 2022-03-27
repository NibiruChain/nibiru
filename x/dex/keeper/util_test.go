package keeper

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestPoolAssetsCoins(t *testing.T) {
	poolAssets := []types.PoolAsset{
		{
			Token: sdk.NewCoin("atom", sdk.NewInt(100)),
		},
		{
			Token: sdk.NewCoin("mtrx", sdk.NewInt(200)),
		},
	}
	coins := PoolAssetsCoins(poolAssets)

	require.Equal(t, coins, sdk.NewCoins(
		sdk.NewCoin("atom", sdk.NewInt(100)),
		sdk.NewCoin("mtrx", sdk.NewInt(200))),
	)
}
