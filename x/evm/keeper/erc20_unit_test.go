package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type UnitSuite struct {
	testutil.LogRoutingSuite
}

func TestUnit(t *testing.T) {
	suite.Run(t, new(UnitSuite))
}

// TestERC20GasLimit - EIP-150: why 63/64 gas forwarding?
//
// Ethereum applies the "63/64 rule" on every CALL. A callee can receive at
// most remainingGas - remainingGas/64. The caller keeps at least 1/64.
// Purpose:
//   - Bound recursion. Forwarded gas shrinks geometrically, so nested calls
//     always terminate.
//   - Preserve caller cleanup. The caller retains gas to handle errors and
//     state after the callee returns.
//   - Prevent griefing. Fixed, overgenerous gas forwarding can enable
//     recursive call bombs that stall execution.
//
// In our module, precompile  ERC20 calls must respect this rule. We compute
// forwarded gas as remaining - remaining/64 and then cap it by an operation
// limit. This test asserts both behaviors:
//  1. With a small cap, we return the cap.
//  2. With a large cap, we return exactly 63/64 of remaining.
//
// It also checks boundary cases (e.g., 64  63) and the infinite meter case
// where remaining == MaxUint64.
func (s *UnitSuite) TestERC20GasLimit() {
	s.Run("getCallGasLimit63_64 respects 63/64 and cap", func() {
		// Arrange a context with a known remaining gas, e.g., 12,800,000
		// (Use a helper that sets GasMeter to a fixed remaining amount.)
		remaining := uint64(12_800_000)
		ctx := sdk.Context{}.WithGasMeter(
			sdk.NewGasMeter(remaining),
		)

		s.Require().Equal(remaining, ctx.GasMeter().GasRemaining())

		// callGas = remaining - floor(remaining/64) = 12,800,000 - 200,000 = 12,600,000
		// then min(callGas, limit)

		// With a small limit:
		s.Equal(uint64(100_000), getCallGasLimit63_64(ctx, 100_000))

		// With a large limit:
		s.Equal(uint64(12_600_000), getCallGasLimit63_64(ctx, 20_000_000))
	})

	s.Run("infinite meter + realistic cap => cap", func() {
		ctx := sdk.Context{}.WithGasMeter(sdk.NewInfiniteGasMeter())
		s.Equal(uint64(100_000), getCallGasLimit63_64(ctx, 100_000))
	})
	s.Run("infinite meter + infinite cap => 63/64 of max u64", func() {
		maxU64 := ^uint64(0) // MaxUint64
		ctx := sdk.Context{}.WithGasMeter(
			sdk.NewGasMeter(maxU64),
		)
		got := getCallGasLimit63_64(ctx, maxU64)
		want := maxU64 - maxU64/64
		s.Equal(
			fmt.Sprintf("%d", want),
			fmt.Sprintf("%d", got),
		)
	})

	s.Run("getCallGasLimit63_64 boundary cases", func() {
		for _, tc := range []struct {
			remaining, limit, expect uint64
		}{
			{64, 1_000_000, 63},    // 64 -> 63
			{63, 1_000_000, 63},    // stays 63
			{1, 1_000_000, 1},      // tiny gas preserved
			{10_000, 5_000, 5_000}, // cap smaller than 63/64
		} {
			ctx := sdk.Context{}.WithGasMeter(sdk.NewGasMeter(tc.remaining))
			s.Equal(tc.expect, getCallGasLimit63_64(ctx, tc.limit))
		}
	})
}
