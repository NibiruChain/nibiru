package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	ammv1 "github.com/MatrixDao/matrix/api/amm"
)

func TestPoolHasEnoughQuoteReserve(t *testing.T) {
	pool := &ammv1.Pool{
		Pair:              "",
		TradeLimitRatio:   "900000",   // 0.9
		QuoteAssetReserve: "10000000", // 10
	}

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
