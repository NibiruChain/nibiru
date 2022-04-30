package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/NibiruChain/nibiru/testutil/keeper"
	"github.com/NibiruChain/nibiru/x/poolincentives/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.PoolincentivesKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
