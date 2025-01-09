// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"time"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

// TestRandom tests the random value generation within the EVM.
func (s *Suite) TestRandom() {
	deps := evmtest.NewTestDeps()
	deployResp, err := evmtest.DeployContract(&deps, embeds.SmartContract_TestRandom)
	s.Require().NoError(err)
	randomContractAddr := deployResp.ContractAddr

	// highjacked LoadERC20BigInt method as it perfectly fits the need of this test
	random1, err := deps.EvmKeeper.LoadERC20BigInt(
		deps.Ctx, embeds.SmartContract_TestRandom.ABI, randomContractAddr, "getRandom",
	)
	s.Require().NoError(err)
	s.Require().NotNil(random1)
	s.Require().NotZero(random1.Int64())

	// Update block time to check that random changes
	deps.Ctx = deps.Ctx.WithBlockTime(deps.Ctx.BlockTime().Add(1 * time.Second))
	random2, err := deps.EvmKeeper.LoadERC20BigInt(
		deps.Ctx, embeds.SmartContract_TestRandom.ABI, randomContractAddr, "getRandom",
	)
	s.Require().NoError(err)
	s.Require().NotNil(random1)
	s.Require().NotZero(random2.Int64())
	s.Require().NotEqual(random1, random2)
}
