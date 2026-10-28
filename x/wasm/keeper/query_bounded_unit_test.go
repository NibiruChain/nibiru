package keeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// TestQuerySmartBounded checks the three promises of QuerySmartBounded with
// the hackatom contract, whose recurse query hashes `work` times and so
// stands in for a contract that loops: a normal query succeeds and reports
// its gas, a query that would exceed the limit returns an error instead of
// panicking, and the caller's gas meter is never charged.
func TestQuerySmartBounded(t *testing.T) {
	contract, ctx, keeper := initRecurseContract(t)
	// Block hooks run with an infinite meter; that is the case to defend.
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())

	t.Run("within the limit", func(t *testing.T) {
		resp, gasUsed, err := keeper.QuerySmartBounded(
			ctx, contract, buildRecurseQuery(t, Recurse{Work: 1}), 3_000_000)
		require.NoError(t, err)
		require.NotEmpty(t, resp)
		require.Positive(t, gasUsed)
		require.Less(t, gasUsed, uint64(3_000_000))
		require.Zero(t, ctx.GasMeter().GasConsumed(),
			"the caller's meter must not be charged")
	})

	t.Run("a looping query runs out of gas without panicking", func(t *testing.T) {
		// About 100 SDK gas per round, so a million rounds need ~100M gas.
		req := buildRecurseQuery(t, Recurse{Work: 1_000_000})
		var (
			resp    []byte
			gasUsed uint64
			err     error
		)
		start := time.Now()
		require.NotPanics(t, func() {
			resp, gasUsed, err = keeper.QuerySmartBounded(ctx, contract, req, 1_000_000)
		})
		t.Logf("out of gas at 1,000,000 gas after %s", time.Since(start))
		require.ErrorContains(t, err, "out of gas")
		require.Nil(t, resp)
		require.Equal(t, uint64(1_000_000), gasUsed)
		require.Zero(t, ctx.GasMeter().GasConsumed())
	})

	t.Run("query error", func(t *testing.T) {
		_, _, err := keeper.QuerySmartBounded(
			ctx, contract, []byte(`{"no_such_query":{}}`), 3_000_000)
		require.Error(t, err)
	})

	t.Run("missing contract", func(t *testing.T) {
		_, _, err := keeper.QuerySmartBounded(
			ctx, RandomAccountAddress(t), buildRecurseQuery(t, Recurse{}), 3_000_000)
		require.Error(t, err)
	})
}
