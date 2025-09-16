package keeper_test

import (
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"

	"github.com/stretchr/testify/suite"
)

// ------------------------------------------------------------
// Test Suite struct definitions

// TestAll: Runs all the tests in the suite.
func TestAll(t *testing.T) {
	suite.Run(t, new(Suite))
	suite.Run(t, new(SuiteFunToken))
}

var (
	_ suite.SetupAllSuite = (*Suite)(nil)
	_ suite.SetupAllSuite = (*SuiteFunToken)(nil)
)

type Suite struct {
	testutil.LogRoutingSuite
}

type SuiteFunToken struct {
	testutil.LogRoutingSuite
}

// ------------------------------------------------------------

// TestIsSimulation verifies the IsSimulation helper function
func (s *Suite) TestIsSimulation() {
	deps := evmtest.NewTestDeps()

	testCases := []struct {
		name     string
		setup    func(ctx sdk.Context) sdk.Context
		expected bool
	}{
		{
			name: "Default context - not simulation",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx
			},
			expected: false,
		},
		{
			name: "Context with simulation=true",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithValue(keeper.SimulationContextKey, true)
			},
			expected: true,
		},
		{
			name: "Context with simulation=false",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithValue(keeper.SimulationContextKey, false)
			},
			expected: false,
		},
		{
			name: "Context with wrong type for simulation key",
			setup: func(ctx sdk.Context) sdk.Context {
				// Set a string instead of bool
				return ctx.WithValue(keeper.SimulationContextKey, "true")
			},
			expected: false,
		},
		{
			name: "Context with nil value",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithValue(keeper.SimulationContextKey, nil)
			},
			expected: false,
		},
		{
			name: "CheckTx context without simulation flag",
			setup: func(ctx sdk.Context) sdk.Context {
				// CheckTx alone doesn't make it a simulation
				return ctx.WithIsCheckTx(true)
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			testCtx := tc.setup(deps.Ctx)
			result := keeper.IsSimulation(testCtx)
			s.Equal(tc.expected, result, "IsSimulation result incorrect for %s", tc.name)
		})
	}
}

// TestIsDeliverTx verifies the isDeliverTx helper function
func (s *Suite) TestIsDeliverTx() {
	deps := evmtest.NewTestDeps()

	testCases := []struct {
		name     string
		setup    func(ctx sdk.Context) sdk.Context
		expected bool
	}{
		{
			name: "Default context is DeliverTx",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx
			},
			expected: true,
		},
		{
			name: "CheckTx context",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithIsCheckTx(true)
			},
			expected: false,
		},
		{
			name: "ReCheckTx context",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithIsReCheckTx(true)
			},
			expected: false,
		},
		{
			name: "Simulation context",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithValue(keeper.SimulationContextKey, true)
			},
			expected: false,
		},
		{
			name: "CheckTx with simulation flag",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithIsCheckTx(true).WithValue(keeper.SimulationContextKey, true)
			},
			expected: false,
		},
		{
			name: "Simulation context with false value",
			setup: func(ctx sdk.Context) sdk.Context {
				// Setting simulation to false should be treated as DeliverTx
				return ctx.WithValue(keeper.SimulationContextKey, false)
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			testCtx := tc.setup(deps.Ctx)
			result := keeper.IsDeliverTx(testCtx)
			s.Equal(tc.expected, result, "IsDeliverTx result incorrect for %s", tc.name)
		})
	}
}

func (s *Suite) TestGetHashFn() {
	deps := evmtest.NewTestDeps()
	fn := deps.EvmKeeper.GetHashFn(deps.Ctx)
	s.Equal(gethcommon.Hash{}, fn(math.MaxInt64+1))
	s.Equal(gethcommon.BytesToHash(deps.Ctx.HeaderHash()), fn(uint64(deps.Ctx.BlockHeight())))
	s.Equal(gethcommon.Hash{}, fn(uint64(deps.Ctx.BlockHeight())+1))
	s.Equal(gethcommon.Hash{}, fn(uint64(deps.Ctx.BlockHeight())-1))
}
