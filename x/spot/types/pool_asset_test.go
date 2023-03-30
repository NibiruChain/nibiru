package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
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

func TestSubtractPoolAssetBalance(t *testing.T) {
	for _, tc := range []struct {
		name          string
		pool          Pool
		tokenDenom    string
		subAmt        sdk.Int
		expectedCoins sdk.Coins
	}{
		{
			name: "subtract liquidity",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 1*common.TO_MICRO),
					},
				},
			},
			tokenDenom:    "aaa",
			subAmt:        sdk.NewInt(1_000),
			expectedCoins: sdk.NewCoins(sdk.NewInt64Coin("aaa", 999_000)),
		},
		{
			name: "subtract all liquidity",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 1*common.TO_MICRO),
					},
				},
			},
			tokenDenom:    "aaa",
			subAmt:        sdk.NewInt(1 * common.TO_MICRO),
			expectedCoins: nil,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.pool.SubtractPoolAssetBalance(tc.tokenDenom, tc.subAmt))
			actualCoins := tc.pool.PoolBalances()
			require.Equal(t, tc.expectedCoins, actualCoins)
		})
	}
}
