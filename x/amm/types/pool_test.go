package types

import (
	ammv1 "github.com/MatrixDao/matrix/api/amm"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestPoolHasEnoughQuoteReserve(t *testing.T) {
	pool := NewPool(
		"BTC:USDM",
		sdk.NewInt(900_000),    // 0.9
		sdk.NewInt(10_000_000), // 10
		sdk.NewInt(10_000_000), // 10
	)

	// less that max ratio
	isEnough, err := PoolHasEnoughQuoteReserve(pool, sdk.NewInt(8_000_000))
	require.NoError(t, err)
	require.True(t, isEnough)

	// equal to ratio limit
	isEnough, err = PoolHasEnoughQuoteReserve(pool, sdk.NewInt(9_000_000))
	require.NoError(t, err)
	require.True(t, isEnough)

	// more than ratio limit
	isEnough, err = PoolHasEnoughQuoteReserve(pool, sdk.NewInt(9_000_001))
	require.NoError(t, err)
	require.False(t, isEnough)
}

func TestGetBaseAmountByQuoteAmount(t *testing.T) {
	pool := NewPool(
		"BTC:USDM",
		sdk.NewInt(900_000),    // 0.9
		sdk.NewInt(10_000_000), // 10
		sdk.NewInt(10_000_000), // 10
	)

	amount := GetBaseAmountByQuoteAmount(ammv1.Direction_ADD_TO_AMM, pool, sdk.NewInt(3_000_000))
}
