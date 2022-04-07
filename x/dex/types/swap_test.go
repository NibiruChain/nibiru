package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCalcOutAmtGivenIn(t *testing.T) {
	for _, tc := range []struct {
		name             string
		pool             Pool
		tokenIn          sdk.Coin
		tokenOutDenom    string
		expectedTokenOut sdk.Coin
	}{
		{
			name: "simple swap",
			pool: Pool{
				PoolParams: PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.0003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 100),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			tokenIn:          sdk.NewInt64Coin("aaa", 10),
			tokenOutDenom:    "bbb",
			expectedTokenOut: sdk.NewInt64Coin("bbb", 9),
		},
		{
			name: "big simple numbers",
			pool: Pool{
				PoolParams: PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.0003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100_000_000),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 100_000_000),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			tokenIn:          sdk.NewInt64Coin("aaa", 10),
			tokenOutDenom:    "bbb",
			expectedTokenOut: sdk.NewInt64Coin("bbb", 9),
		},
		{
			name: "big simple numbers, huge swap fee",
			pool: Pool{
				PoolParams: PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.5"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1_000_000),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1_000_000),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			tokenIn:          sdk.NewInt64Coin("aaa", 10),
			tokenOutDenom:    "bbb",
			expectedTokenOut: sdk.NewInt64Coin("bbb", 4),
		},
		{
			name: "real numbers",
			pool: Pool{
				PoolParams: PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.0003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 3498723457),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 23318504),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			tokenIn:       sdk.NewInt64Coin("aaa", 5844683),
			tokenOutDenom: "bbb",
			// solved with wolfram alpha (https://www.wolframalpha.com/input?i=23318504+-+%283498723457*23318504%29%2F+%283498723457%2B5844683*%281-0.0003%29%29)
			expectedTokenOut: sdk.NewInt64Coin("bbb", 38877),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tokenOut, err := tc.pool.CalcOutAmtGivenIn(tc.tokenIn, tc.tokenOutDenom)
			require.NoError(t, err)
			require.Equal(t, tc.expectedTokenOut, tokenOut)
		})
	}
}

func TestCalcInAmtGivenOut(t *testing.T) {
	for _, tc := range []struct {
		name            string
		pool            Pool
		tokenOut        sdk.Coin
		tokenInDenom    string
		expectedTokenIn sdk.Coin
	}{
		{
			name: "simple swap",
			pool: Pool{
				PoolParams: PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.0003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 100),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			tokenOut:        sdk.NewInt64Coin("bbb", 9),
			tokenInDenom:    "aaa",
			expectedTokenIn: sdk.NewInt64Coin("aaa", 10),
		},
		{
			name: "big simple numbers",
			pool: Pool{
				PoolParams: PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.0003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100_000_000),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 100_000_000),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			tokenOut:        sdk.NewInt64Coin("bbb", 9),
			tokenInDenom:    "aaa",
			expectedTokenIn: sdk.NewInt64Coin("aaa", 10),
		},
		{
			name: "big simple numbers, huge swap fee",
			pool: Pool{
				PoolParams: PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.5"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1_000_000),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1_000_000),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			tokenOut:        sdk.NewInt64Coin("bbb", 4),
			tokenInDenom:    "aaa",
			expectedTokenIn: sdk.NewInt64Coin("aaa", 9),
		},
		{
			name: "real numbers",
			pool: Pool{
				PoolParams: PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.0003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 3498723457),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 23318504),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			// solved with wolfram alpha (https://www.wolframalpha.com/input?i=%28%283498723457*23318504%29%2F+%2823318504-38877%29+-+3498723457%29%2F%281-0.0003%29)
			tokenOut:        sdk.NewInt64Coin("bbb", 38877),
			tokenInDenom:    "aaa",
			expectedTokenIn: sdk.NewInt64Coin("aaa", 5844626),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// CalcInAmtGivenOut is the inverse, so we can use the same test inputs/outputs
			tokenIn, err := tc.pool.CalcInAmtGivenOut(tc.tokenOut, tc.tokenInDenom)
			require.NoError(t, err)
			require.Equal(t, tc.expectedTokenIn, tokenIn)
		})
	}
}
