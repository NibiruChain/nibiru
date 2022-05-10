package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCreatePool(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t)

	vpoolKeeper.CreatePool(
		ctx,
		NUSDPair,
		sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
		sdk.NewInt(10_000_000),       // 10 tokens
		sdk.NewInt(5_000_000),        // 5 tokens
		sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
	)

	exists := vpoolKeeper.existsPool(ctx, NUSDPair)
	require.True(t, exists)

	notExist := vpoolKeeper.existsPool(ctx, "BTC:OTHER")
	require.False(t, notExist)
}
