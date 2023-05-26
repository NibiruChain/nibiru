package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMaximalSharesFromExactRatioJoin(t *testing.T) {
	for _, tc := range []struct {
		name              string
		poolAssets        []PoolAsset
		existingShares    int64
		tokensIn          sdk.Coins
		expectedNumShares sdkmath.Int
		expectedRemCoins  sdk.Coins
	}{
		{
			name: "all coins deposited",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 100),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedNumShares: sdk.NewInt(100),
			expectedRemCoins:  sdk.NewCoins(),
		},
		{
			name: "some coins deposited",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 100),
				sdk.NewInt64Coin("bbb", 50),
			),
			expectedNumShares: sdk.NewInt(50),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
			),
		},
		{
			name: "limited by smallest amount",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 1),
				sdk.NewInt64Coin("bbb", 50),
			),
			expectedNumShares: sdk.NewInt(1),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("bbb", 49),
			),
		},
		{
			name: "limited by smallest amount - 2",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 200),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 10),
			),
			expectedNumShares: sdk.NewInt(5),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 5),
			),
		},
		{
			name: "right number of LP shares",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 50),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			existingShares: 150,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 50),
			),
			expectedNumShares: sdk.NewInt(75),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 25),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := Pool{
				Id:          1,
				Address:     "some_address",
				PoolParams:  PoolParams{},
				PoolAssets:  tc.poolAssets,
				TotalWeight: sdk.OneInt(),
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", tc.existingShares),
			}
			numShares, remCoins, _ := pool.numSharesOutFromTokensIn(tc.tokensIn)
			require.Equal(t, tc.expectedNumShares, numShares)
			require.Equal(t, tc.expectedRemCoins, remCoins)
		})
	}
}

func TestTokensOutFromExactSharesHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name              string
		pool              Pool
		numSharesIn       sdkmath.Int
		expectedTokensOut sdk.Coins
		expectedFees      sdk.Coins
	}{
		{
			name: "all coins withdrawn, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
				PoolParams: PoolParams{
					ExitFee: sdk.ZeroDec(),
				},
			},
			numSharesIn: sdk.NewInt(50),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 200),
			),
			expectedFees: sdk.Coins{},
		},
		{
			name: "partial coins withdrawn, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
				PoolParams: PoolParams{
					ExitFee: sdk.ZeroDec(),
				},
			},
			numSharesIn: sdk.NewInt(25),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 100),
			),
			expectedFees: sdk.Coins{},
		},
		{
			name: "fractional coins withdrawn truncates to int, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1000),
				PoolParams: PoolParams{
					ExitFee: sdk.ZeroDec(),
				},
			},
			numSharesIn: sdk.NewInt(25),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 2),
				sdk.NewInt64Coin("foo", 5),
			),
			expectedFees: sdk.Coins{},
		},
		{
			name: "all coins withdrawn, with exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
				PoolParams: PoolParams{
					ExitFee: sdk.MustNewDecFromStr("0.5"),
				},
			},
			numSharesIn: sdk.NewInt(50),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 100),
			),
			expectedFees: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 100),
			),
		},
		{
			name: "partial coins withdrawn, with exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
				PoolParams: PoolParams{
					ExitFee: sdk.MustNewDecFromStr("0.5"),
				},
			},
			numSharesIn: sdk.NewInt(25),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 25),
				sdk.NewInt64Coin("foo", 50),
			),
			expectedFees: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 25),
				sdk.NewInt64Coin("foo", 50),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tokensOut, fees, err := tc.pool.TokensOutFromPoolSharesIn(tc.numSharesIn)
			require.NoError(t, err)
			require.Equal(t, tc.expectedTokensOut, tokensOut)
			require.Equal(t, tc.expectedFees, fees)
		})
	}
}

func TestTokensOutFromExactSharesErrors(t *testing.T) {
	for _, tc := range []struct {
		name        string
		pool        Pool
		numSharesIn sdkmath.Int
	}{
		{
			name: "zero pool shares",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
			},
			numSharesIn: sdk.NewInt(0),
		},
		{
			name: "too many pool shares",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
			},
			numSharesIn: sdk.NewInt(51),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := tc.pool.TokensOutFromPoolSharesIn(tc.numSharesIn)
			require.Error(t, err)
		})
	}
}

func TestUpdateLiquidityHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		pool                  Pool
		numShares             sdkmath.Int
		newLiquidity          sdk.Coins
		expectedNumShares     sdkmath.Int
		expectedNewPoolAssets []PoolAsset
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
			},
			numShares: sdk.NewInt(10),
			newLiquidity: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 20),
			),
			expectedNumShares: sdk.NewInt(110),
			expectedNewPoolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 110),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 220),
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pool.incrementBalances(tc.numShares, tc.newLiquidity)
			require.NoError(t, err)
			require.Equal(t, tc.expectedNumShares, tc.pool.TotalShares.Amount)
			require.Equal(t, tc.expectedNewPoolAssets, tc.pool.PoolAssets)
		})
	}
}

func TestUpdateLiquidityInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name         string
		pool         Pool
		numShares    sdkmath.Int
		newLiquidity sdk.Coins
	}{
		{
			name: "add non-existent coin",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			numShares: sdk.NewInt(10),
			newLiquidity: sdk.NewCoins(
				sdk.NewInt64Coin("bbb", 20),
			),
		},
		{
			name: "no existing liquidity",
			pool: Pool{
				PoolAssets:  []PoolAsset{},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			numShares: sdk.NewInt(10),
			newLiquidity: sdk.NewCoins(
				sdk.NewInt64Coin("bbb", 20),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pool.incrementBalances(tc.numShares, tc.newLiquidity)
			require.Error(t, err)
		})
	}
}

func TestNumSharesOutStableswap(t *testing.T) {
	for _, tc := range []struct {
		name              string
		poolAssets        []PoolAsset
		existingShares    int64
		tokensIn          sdk.Coins
		expectedNumShares sdkmath.Int
		A                 sdkmath.Int
		expectedError     error
	}{
		{
			name: "all coins deposited",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			A:              sdk.NewInt(2000),
			existingShares: 50,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 100),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedNumShares: sdk.NewInt(50),
		},
		{
			name: "all coins deposited - 3pool",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
				{
					Token: sdk.NewInt64Coin("ccc", 100),
				},
			},
			A:              sdk.NewInt(2000),
			existingShares: 50,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 100),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedNumShares: sdk.NewInt(33),
		},
		{
			name: "all coins deposited - imbalance",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			A:              sdk.NewInt(2000),
			existingShares: 50,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedNumShares: sdk.NewInt(37),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := Pool{
				Id:          1,
				Address:     "some_address",
				PoolParams:  PoolParams{A: tc.A},
				PoolAssets:  tc.poolAssets,
				TotalWeight: sdk.OneInt(),
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", tc.existingShares),
			}
			numShares, err := pool.numSharesOutFromTokensInStableSwap(tc.tokensIn)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedNumShares, numShares)
			}
		})
	}
}
