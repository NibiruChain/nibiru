package evmstate_test

import (
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"

	"github.com/stretchr/testify/suite"
)

// ------------------------------------------------------------
// Test Suite struct definitions

// TestAll: Runs all the tests in the suite.
func TestEvmState(t *testing.T) {
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
				return ctx.WithValue(evm.CtxKeyEvmSimulation, true)
			},
			expected: true,
		},
		{
			name: "Context with simulation=false",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithValue(evm.CtxKeyEvmSimulation, false)
			},
			expected: false,
		},
		{
			name: "Context with wrong type for simulation key",
			setup: func(ctx sdk.Context) sdk.Context {
				// Set a string instead of bool
				return ctx.WithValue(evm.CtxKeyEvmSimulation, "true")
			},
			expected: false,
		},
		{
			name: "Context with nil value",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithValue(evm.CtxKeyEvmSimulation, nil)
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
			testCtx := tc.setup(deps.Ctx())
			result := evmstate.IsSimulation(testCtx)
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
				return ctx.WithValue(evm.CtxKeyEvmSimulation, true)
			},
			expected: false,
		},
		{
			name: "CheckTx with simulation flag",
			setup: func(ctx sdk.Context) sdk.Context {
				return ctx.WithIsCheckTx(true).WithValue(evm.CtxKeyEvmSimulation, true)
			},
			expected: false,
		},
		{
			name: "Simulation context with false value",
			setup: func(ctx sdk.Context) sdk.Context {
				// Setting simulation to false should be treated as DeliverTx
				return ctx.WithValue(evm.CtxKeyEvmSimulation, false)
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			testCtx := tc.setup(deps.Ctx())
			result := evmstate.IsDeliverTx(testCtx)
			s.Equal(tc.expected, result, "IsDeliverTx result incorrect for %s", tc.name)
		})
	}
}

func (s *Suite) TestGetHashFn() {
	deps := evmtest.NewTestDeps()
	fn := deps.EvmKeeper.GetHashFn(deps.Ctx())
	s.Equal(gethcommon.Hash{}, fn(math.MaxInt64+1))
	s.Equal(gethcommon.BytesToHash(deps.Ctx().HeaderHash()), fn(uint64(deps.Ctx().BlockHeight())))
	s.Equal(gethcommon.Hash{}, fn(uint64(deps.Ctx().BlockHeight())+1))
	s.Equal(gethcommon.Hash{}, fn(uint64(deps.Ctx().BlockHeight())-1))
}

func (s *Suite) TestUpdateParams() {
	deps := evmtest.NewTestDeps()

	s.Run("happy path: with permission and valid input", func() {
		authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		params := evm.DefaultParams()

		resp, err := deps.EvmKeeper.UpdateParams(deps.GoCtx(), &evm.MsgUpdateParams{
			Authority: authority,
			Params:    params,
		})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		got := deps.EvmKeeper.GetParams(deps.Ctx())
		s.Require().Equal(params.CreateFuntokenFee, got.CreateFuntokenFee)
		s.Require().Equal(params.CanonicalWnibi.Hex(), got.CanonicalWnibi.Hex())
	})

	s.Run("sad path: no permission", func() {
		authority := deps.Sender.NibiruAddr.String()
		params := evm.DefaultParams()

		_, err := deps.EvmKeeper.UpdateParams(deps.GoCtx(), &evm.MsgUpdateParams{
			Authority: authority,
			Params:    params,
		})
		s.Require().Error(err)
		s.Require().ErrorContains(err, "invalid signing authority")
	})
}
