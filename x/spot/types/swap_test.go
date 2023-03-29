package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
)

func TestCalcOutAmtGivenIn(t *testing.T) {
	for _, tc := range []struct {
		name             string
		pool             Pool
		tokenIn          sdk.Coin
		tokenOutDenom    string
		expectedTokenOut sdk.Coin
		expectedFee      sdk.Coin
		shouldError      bool
	}{
		{
			name: "simple swap",
			pool: Pool{
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					SwapFee:  sdk.MustNewDecFromStr("0.0003"),
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
			expectedFee:      sdk.NewInt64Coin("aaa", 0), // 0.0003 * 10 = 0.003, truncated to 0
		},
		{
			name: "big simple numbers",
			pool: Pool{
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					SwapFee:  sdk.MustNewDecFromStr("0.0003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100*common.MICRO),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 100*common.MICRO),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			tokenIn:          sdk.NewInt64Coin("aaa", 10),
			tokenOutDenom:    "bbb",
			expectedTokenOut: sdk.NewInt64Coin("bbb", 9),
			expectedFee:      sdk.NewInt64Coin("aaa", 0), // 0.0003 * 10 = 0.003, truncated to 0
		},
		{
			name: "big simple numbers, huge swap fee",
			pool: Pool{
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					SwapFee:  sdk.MustNewDecFromStr("0.5"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1*common.MICRO),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1*common.MICRO),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
			},
			tokenIn:          sdk.NewInt64Coin("aaa", 10),
			tokenOutDenom:    "bbb",
			expectedTokenOut: sdk.NewInt64Coin("bbb", 4),
			expectedFee:      sdk.NewInt64Coin("aaa", 5), // 0.5 * 10 = 5
		},
		{
			name: "real numbers",
			pool: Pool{
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					SwapFee:  sdk.MustNewDecFromStr("0.0003"),
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
			expectedFee:      sdk.NewInt64Coin("aaa", 1753), // 0.0003 * 5844683 = 1753.4049, truncated to 1753
		},
		{
			name: "swap with very low output token amount",
			pool: Pool{
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					SwapFee:  sdk.MustNewDecFromStr("0.0003"),
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
			tokenIn:       sdk.NewInt64Coin("aaa", 1),
			tokenOutDenom: "bbb",
			shouldError:   true,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tokenOut, fee, err := tc.pool.CalcOutAmtGivenIn(tc.tokenIn, tc.tokenOutDenom, false)
			if tc.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedTokenOut, tokenOut)
				require.Equal(t, tc.expectedFee, fee)
			}
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
					PoolType: PoolType_BALANCER,
					SwapFee:  sdk.MustNewDecFromStr("0.0003"),
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
					PoolType: PoolType_BALANCER,
					SwapFee:  sdk.MustNewDecFromStr("0.0003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100*common.MICRO),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 100*common.MICRO),
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
					PoolType: PoolType_BALANCER,
					SwapFee:  sdk.MustNewDecFromStr("0.5"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1*common.MICRO),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1*common.MICRO),
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
					PoolType: PoolType_BALANCER,
					SwapFee:  sdk.MustNewDecFromStr("0.0003"),
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

func TestApplySwap(t *testing.T) {
	for _, tc := range []struct {
		name               string
		pool               Pool
		tokenIn            sdk.Coin
		tokenOut           sdk.Coin
		expectedPoolAssets []PoolAsset
		shouldError        bool
	}{
		{
			name: "apply simple swap",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
			},
			tokenIn:  sdk.NewInt64Coin("aaa", 50),
			tokenOut: sdk.NewInt64Coin("bbb", 75),
			expectedPoolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 150),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 125),
				},
			},
			shouldError: false,
		},
		{
			name: "swap fails due to too large numbers",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
			},
			tokenIn:     sdk.NewInt64Coin("aaa", 1),
			tokenOut:    sdk.NewInt64Coin("bbb", 201),
			shouldError: true,
		},
		{
			name: "swap fails due to zero tokenIn",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
			},
			tokenIn:     sdk.NewInt64Coin("aaa", 0),
			tokenOut:    sdk.NewInt64Coin("bbb", 100),
			shouldError: true,
		},
		{
			name: "swap fails due to zero tokenOut",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
			},
			tokenIn:     sdk.NewInt64Coin("aaa", 100),
			tokenOut:    sdk.NewInt64Coin("bbb", 0),
			shouldError: true,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pool.ApplySwap(tc.tokenIn, tc.tokenOut)
			if tc.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedPoolAssets, tc.pool.PoolAssets)
			}
		})
	}
}
