package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestPoolAssetValidateError(t *testing.T) {
	tests := []struct {
		name   string
		pa     PoolAsset
		errMsg string
	}{
		{
			name: "coin amount too little",
			pa: PoolAsset{
				Token:  sdk.NewInt64Coin("foo", 0),
				Weight: sdk.NewInt(1),
			},
			errMsg: "can't add the zero or negative balance of token",
		},
		{
			name: "weight too little",
			pa: PoolAsset{
				Token:  sdk.NewInt64Coin("foo", 1),
				Weight: sdk.NewInt(0),
			},
			errMsg: "can't add the zero or negative balance of token",
		},
		{
			name: "weight too high",
			pa: PoolAsset{
				Token:  sdk.NewInt64Coin("foo", 1),
				Weight: sdk.NewInt(1 << 50),
			},
			errMsg: "a token's weight in the pool must be less than 1^50",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require.Errorf(t, tc.pa.Validate(), tc.errMsg)
		})
	}

}

func TestPoolAssetValidateSuccess(t *testing.T) {
	tests := []struct {
		name string
		pa   PoolAsset
	}{
		{
			name: "successful validation",
			pa: PoolAsset{
				Token:  sdk.NewInt64Coin("foo", 1),
				Weight: sdk.NewInt(1),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.pa.Validate())
		})
	}

}

func TestPoolAssetGetLiquidity(t *testing.T) {
	for _, tc := range []struct {
		name              string
		poolAssets        []PoolAsset
		expectedLiquidity sdk.Coins
	}{
		{
			name: "get liquidity",
			poolAssets: []PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1),
					Weight: sdk.NewInt(1),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 2),
					Weight: sdk.NewInt(1),
				},
				{
					Token:  sdk.NewInt64Coin("ccc", 3),
					Weight: sdk.NewInt(1),
				},
				{
					Token:  sdk.NewInt64Coin("ddd", 4),
					Weight: sdk.NewInt(1),
				},
			},
			expectedLiquidity: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 1),
				sdk.NewInt64Coin("bbb", 2),
				sdk.NewInt64Coin("ccc", 3),
				sdk.NewInt64Coin("ddd", 4),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedLiquidity, GetPoolLiquidity(tc.poolAssets))
		})
	}

}
