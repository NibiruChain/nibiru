package keeper_test

import (
	"testing"

	testkeeper "github.com/MatrixDao/matrix/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestGetAndSetNextPoolNumber(t *testing.T) {
	k, ctx, _, _ := testkeeper.DexKeeper(t)

	k.SetNextPoolNumber(ctx, 100)
	poolNumber := k.GetNextPoolNumber(ctx)

	require.EqualValues(t, poolNumber, 100)
}

func TestGetNextPoolNumberAndIncrement(t *testing.T) {
	k, ctx, _, _ := testkeeper.DexKeeper(t)

	k.SetNextPoolNumber(ctx, 200)

	poolNumber := k.GetNextPoolNumberAndIncrement(ctx)
	require.EqualValues(t, poolNumber, 200)

	poolNumber = k.GetNextPoolNumber(ctx)
	require.EqualValues(t, poolNumber, 201)
}
