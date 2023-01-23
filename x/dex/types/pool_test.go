package types

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGetPoolShareBaseDenom(t *testing.T) {
	require.Equal(t, "nibiru/pool/123", GetPoolShareBaseDenom(123))
}

func TestGetPoolShareDisplayDenom(t *testing.T) {
	require.Equal(t, "NIBIRU-POOL-123", GetPoolShareDisplayDenom(123))
}

func TestGetAddress(t *testing.T) {
	tests := []struct {
		name        string
		pool        Pool
		expectPanic bool
	}{
		{
			name: "empty address",
			pool: Pool{
				Address: "",
			},
			expectPanic: true,
		},
		{
			name: "invalid address",
			pool: Pool{
				Address: "asdf",
			},
			expectPanic: true,
		},
		{
			name: "valid address",
			pool: Pool{
				Address: testutil.AccAddress().String(),
			},
			expectPanic: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				require.Panics(t, func() {
					tc.pool.GetAddress()
				})
			} else {
				require.NotPanics(t, func() {
					tc.pool.GetAddress()
				})
			}
		})
	}
}

func TestNewPool(t *testing.T) {
	poolAccountAddr := testutil.AccAddress()
	poolParams := PoolParams{
		PoolType: PoolType_BALANCER,
		SwapFee:  sdk.NewDecWithPrec(3, 2),
		ExitFee:  sdk.NewDecWithPrec(3, 2),
	}
	poolAssets := []PoolAsset{
		{
			Token:  sdk.NewInt64Coin("foo", 100),
			Weight: sdk.NewInt(1),
		},
		{
			Token:  sdk.NewInt64Coin("bar", 100),
			Weight: sdk.NewInt(1),
		},
	}

	pool, err := NewPool(1 /*=poold*/, poolAccountAddr, poolParams, poolAssets)
	require.NoError(t, err)
	require.Equal(t, Pool{
		Id:         1,
		Address:    poolAccountAddr.String(),
		PoolParams: poolParams,
		PoolAssets: []PoolAsset{
			{
				Token:  sdk.NewInt64Coin("bar", 100),
				Weight: sdk.NewInt(1 << 30),
			},
			{
				Token:  sdk.NewInt64Coin("foo", 100),
				Weight: sdk.NewInt(1 << 30),
			},
		},
		TotalWeight: sdk.NewInt(2 << 30),
		TotalShares: sdk.NewCoin("nibiru/pool/1", sdk.NewIntWithDecimal(100, 18)),
	}, pool)
}

func TestJoinPoolHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name              string
		pool              Pool
		tokensIn          sdk.Coins
		expectedNumShares sdk.Int
		expectedRemCoins  sdk.Coins
		expectedPool      Pool
	}{
		{
			name: "all coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_BALANCER},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 20),
			),
			expectedNumShares: sdk.NewInt(10),
			expectedRemCoins:  sdk.NewCoins(),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 110),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 220),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 110),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_BALANCER},
			},
		},
		{
			name: "all coins deposited - stableswap",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_STABLESWAP},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 20),
			),
			expectedNumShares: sdk.NewInt(10),
			expectedRemCoins:  sdk.NewCoins(),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 110),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 220),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 110),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_STABLESWAP},
			},
		},
		{
			name: "partial coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_BALANCER},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 10),
			),
			expectedNumShares: sdk.NewInt(5),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 5),
			),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 105),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 210),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 105),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_BALANCER},
			},
		},
		{
			name: "difficult numbers",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 3_498_579),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 1_403_945),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1*common.Precision),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_BALANCER},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 4859), // 0.138885 % of pool
				sdk.NewInt64Coin("bbb", 1345), // 0.09580147 % of pool
			),
			expectedNumShares: sdk.NewInt(958),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 1507),
			),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 3_501_931),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 1_405_290),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1_000_958),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_BALANCER},
			},
		},
		{
			name: "difficult numbers ~ stableswap",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 3_498_579),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 1_403_945),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1*common.Precision),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_STABLESWAP},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 4859), // 0.138885 % of pool
				sdk.NewInt64Coin("bbb", 1345), // 0.09580147 % of pool
			),
			expectedRemCoins:  sdk.NewCoins(),
			expectedNumShares: sdk.NewInt(1264),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 3_503_438),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 1_405_290),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1_001_264),
				PoolParams:  PoolParams{A: sdk.NewInt(100), PoolType: PoolType_STABLESWAP},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			numShares, remCoins, err := tc.pool.AddTokensToPool(tc.tokensIn)
			require.NoError(t, err)
			require.Equal(t, tc.expectedNumShares, numShares)
			require.Equal(t, tc.expectedRemCoins, remCoins)
			require.Equal(t, tc.expectedPool, tc.pool)
		})
	}
}

func TestJoinPoolAllTokens(t *testing.T) {
	for _, tc := range []struct {
		name              string
		pool              Pool
		tokensIn          sdk.Coins
		expectedNumShares sdk.Int
		expectedRemCoins  sdk.Coins
		expectedPool      Pool
	}{
		{
			name: "all coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 200),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{PoolType: PoolType_BALANCER, SwapFee: sdk.ZeroDec()},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 20),
			),
			expectedNumShares: sdk.NewInt(10),
			expectedRemCoins:  sdk.NewCoins(),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 110),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 220),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 110),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{PoolType: PoolType_BALANCER, SwapFee: sdk.ZeroDec()},
			},
		},
		{
			name: "partial coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 200),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{PoolType: PoolType_BALANCER, SwapFee: sdk.ZeroDec()},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 10),
			),
			expectedNumShares: sdk.NewInt(7),
			expectedRemCoins:  sdk.NewCoins(),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 110),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 210),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 107),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{PoolType: PoolType_BALANCER, SwapFee: sdk.ZeroDec()},
			},
		},
		{
			name: "difficult numbers",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 3_498_579),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1_403_945),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1*common.Precision),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{PoolType: PoolType_BALANCER, SwapFee: sdk.ZeroDec()},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 4_859), // 0.138885 % of pool
				sdk.NewInt64Coin("bbb", 1_345), // 0.09580147 % of pool
			),
			expectedNumShares: sdk.NewInt(1173),
			expectedRemCoins:  sdk.NewCoins(),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 3_503_438),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1_405_290),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1_001_173),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{PoolType: PoolType_BALANCER, SwapFee: sdk.ZeroDec()},
			},
		},
		{
			name: "difficult numbers - single asset join",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 3_498_579),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1_403_945),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1*common.Precision),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{PoolType: PoolType_BALANCER, SwapFee: sdk.ZeroDec()},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 4_859), // 0.138885 % of pool
			),
			expectedNumShares: sdk.NewInt(694),
			expectedRemCoins:  sdk.NewCoins(),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 3_503_438),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1_403_945),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1_000_694),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{PoolType: PoolType_BALANCER, SwapFee: sdk.ZeroDec()},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			numShares, remCoins, err := tc.pool.AddAllTokensToPool(tc.tokensIn)
			require.NoError(t, err)
			require.Equal(t, tc.expectedNumShares, numShares)
			require.Equal(t, tc.expectedRemCoins, remCoins)
			require.Equal(t, tc.expectedPool, tc.pool)
		})
	}
}

func TestExitPoolHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		pool                    Pool
		exitingShares           sdk.Coin
		expectedCoins           sdk.Coins
		expectedRemainingShares sdk.Coin
		expectedExitedCoins     sdk.Coins
	}{
		{
			name: "all coins withdrawn, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					ExitFee:  sdk.ZeroDec(),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 100),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 0),
			expectedCoins:           nil,
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 100),
				sdk.NewInt64Coin("bbb", 200),
			),
		},
		{
			name: "all coins withdrawn, exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					ExitFee:  sdk.MustNewDecFromStr("0.5"),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 100),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 0),
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 100),
			),
		},
		{
			name: "some coins withdrawn, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					ExitFee:  sdk.ZeroDec(),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 50),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 100),
			),
		},
		{
			name: "some coins withdrawn, exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					ExitFee:  sdk.MustNewDecFromStr("0.5"),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 50),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 75),
				sdk.NewInt64Coin("bbb", 150),
			),
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 25),
				sdk.NewInt64Coin("bbb", 50),
			),
		},
		{
			name: "real numbers",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 34_586_245),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 65_469_884),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 2_347_652),
				PoolParams: PoolParams{
					PoolType: PoolType_BALANCER,
					ExitFee:  sdk.MustNewDecFromStr("0.003"),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 74_747),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 2_272_905),
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 33_488_356),
				sdk.NewInt64Coin("bbb", 63_391_639),
			),
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 1_097_889),
				sdk.NewInt64Coin("bbb", 2_078_245),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			exitedCoins, err := tc.pool.ExitPool(tc.exitingShares.Amount)
			require.NoError(t, err)
			require.Equal(t, tc.expectedCoins, tc.pool.PoolBalances())
			// Comparing zero initialized sdk.Int with zero value sdk.Int leads to different results
			if tc.expectedRemainingShares.IsZero() {
				require.True(t, tc.pool.TotalShares.IsZero())
			} else {
				require.Equal(t, tc.expectedRemainingShares, tc.pool.TotalShares)
			}
			require.Equal(t, tc.expectedExitedCoins, exitedCoins)
		})
	}
}

// helper function to create dummy test pools
func MockPool(assets []PoolAsset) Pool {
	return Pool{
		Id: 1,
		PoolParams: PoolParams{
			PoolType: PoolType_BALANCER,
			SwapFee:  sdk.SmallestDec(),
			ExitFee:  sdk.SmallestDec(),
		},
		PoolAssets:  assets,
		TotalShares: sdk.NewInt64Coin(GetPoolShareBaseDenom(1), 100),
		TotalWeight: sdk.NewInt(2),
	}
}

func TestUpdatePoolAssetTokens(t *testing.T) {
	for _, tc := range []struct {
		name               string
		poolAssets         []PoolAsset
		newAssets          []sdk.Coin
		expectedPoolAssets []PoolAsset
	}{
		{
			name: "update pool asset balances",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 200),
				},
			},
			newAssets: []sdk.Coin{
				sdk.NewInt64Coin("aaa", 150),
				sdk.NewInt64Coin("bbb", 125),
			},
			expectedPoolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 150),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 125),
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := MockPool(tc.poolAssets)
			require.NoError(t, pool.updatePoolAssetBalances(tc.newAssets...))
			require.Equal(t, tc.expectedPoolAssets, pool.PoolAssets)
		})
	}
}

func TestGetD(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		poolAssets             []PoolAsset
		amplificationParameter sdk.Int
		expectedErr            error
		expectedD              uint64
	}{
		{
			name: "Compute D - 3 assets - tested against Curve contracts code..",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 200),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
				{
					Token: sdk.NewInt64Coin("ccc", 100),
				},
			},
			amplificationParameter: sdk.NewInt(1),
			expectedErr:            nil,
			expectedD:              397,
		},
		{
			name: "Compute D - 2 assets - tested against Curve contracts code..",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 200),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			amplificationParameter: sdk.NewInt(1),
			expectedErr:            nil,
			expectedD:              294,
		},
		{
			name: "Compute D - 2 assets, A big - tested against Curve contracts code..",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 200),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			amplificationParameter: sdk.NewInt(4000),
			expectedErr:            nil,
			expectedD:              299,
		},
		{
			name: "Compute D - 2 assets, A big, high values - tested against Curve contracts code..",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 200*common.Precision),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100*common.Precision),
				},
			},
			amplificationParameter: sdk.NewInt(4000),
			expectedErr:            nil,
			expectedD:              299997656,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := Pool{
				PoolAssets: tc.poolAssets,
				PoolParams: PoolParams{A: tc.amplificationParameter},
			}

			D, err := pool.GetD(pool.PoolAssets)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedD, D.Uint64())
		})
	}
}

type TestCaseDy struct {
	balance       []uint64
	amplification sdk.Int
	send          int
	receive       int
	dx            sdk.Int
	expectedDy    sdk.Int
}

/*
createTestCases reads the data from the csv file generated with the python curve model and load them into a TestCaseDy
object.

Columns schema of the file:
  - balances: the balance of the pool
  - amplification: the amplification parameter for the pool
  - send: the id of the token sent to the pool for the swap
  - recv: the id of the token expected out of the pool
  - dx: the number of token sent for the swap
  - dy: the expected number of token from the curve python model.
*/
func createTestCases(data [][]string) (testCases []TestCaseDy, err error) {
	for i, line := range data {
		if i > 0 { // omit header line
			var rec TestCaseDy

			err := json.Unmarshal([]byte(line[0]), &rec.balance)
			if err != nil {
				return testCases, err
			}

			amplification, err := strconv.ParseInt(line[1], 10, 64)
			if err != nil {
				return testCases, err
			}

			rec.amplification = sdk.NewInt(amplification)

			rec.send, _ = strconv.Atoi(line[2])
			rec.receive, _ = strconv.Atoi(line[3])

			dx, err := strconv.ParseInt(line[4], 10, 64)
			if err != nil {
				return testCases, err
			}

			expectedDy, err := strconv.ParseInt(line[5], 10, 64)
			if err != nil {
				return testCases, err
			}

			rec.dx = sdk.NewInt(dx)
			rec.expectedDy = sdk.NewInt(expectedDy)

			testCases = append(testCases, rec)
		}
	}
	return testCases, nil
}

func TestSolveStableswapInvariant(t *testing.T) {
	t.Run("Test csv file", func(t *testing.T) {
		f, err := os.Open("misc/stabletests.csv")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		// read csv values using csv.Reader
		csvReader := csv.NewReader(f)
		data, err := csvReader.ReadAll()
		if err != nil {
			log.Fatal(err)
		}

		testCases, err := createTestCases(data)
		if err != nil {
			log.Fatal(err)
		}

		for _, tc := range testCases {
			tc := tc

			var poolAssets []PoolAsset

			for i, balance := range tc.balance {
				poolAssets = append(poolAssets, PoolAsset{Token: sdk.NewCoin("token"+strconv.Itoa(i), sdk.NewIntFromUint64(balance))})
			}

			pool := Pool{
				PoolAssets: poolAssets,
				PoolParams: PoolParams{A: tc.amplification},
			}
			denomIn := "token" + strconv.Itoa(tc.send)
			denomOut := "token" + strconv.Itoa(tc.receive)

			_, poolAssetIn, err := pool.getPoolAssetAndIndex(denomIn)
			require.NoError(t, err)

			dy, err := pool.SolveStableswapInvariant(
				/* tokenIn = */ sdk.NewCoin(denomIn, tc.dx).Add(poolAssetIn.Token),
				/* tokenOutDenom = */ denomOut,
			)
			require.NoError(t, err)

			_, poolAssetOut, err := pool.getPoolAssetAndIndex(denomOut)
			require.NoError(t, err)

			require.NoError(t, err)
			require.Equal(t, tc.expectedDy, poolAssetOut.Token.Amount.Sub(dy))
		}
	})
}
