package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeeper_GetVoteTargets(t *testing.T) {
	input := CreateTestInput(t)

	input.OracleKeeper.ClearPairs(input.Ctx)

	expectedTargets := []string{"bar", "foo", "whoowhoo"}
	for _, target := range expectedTargets {
		input.OracleKeeper.SetPair(input.Ctx, target)
	}

	targets := input.OracleKeeper.GetVoteTargets(input.Ctx)
	require.Equal(t, expectedTargets, targets)
}

func TestKeeper_IsVoteTarget(t *testing.T) {
	input := CreateTestInput(t)

	input.OracleKeeper.ClearPairs(input.Ctx)

	validTargets := []string{"bar", "foo", "whoowhoo"}
	for _, target := range validTargets {
		input.OracleKeeper.SetPair(input.Ctx, target)
		require.True(t, input.OracleKeeper.IsVoteTarget(input.Ctx, target))
	}
}
