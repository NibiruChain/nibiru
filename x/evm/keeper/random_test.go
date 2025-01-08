// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
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
	resp, err := deps.EvmKeeper.LoadERC20BigInt(
		deps.Ctx, embeds.SmartContract_TestRandom.ABI, randomContractAddr, "getRandom",
	)
	s.Require().NoError(err)
	s.Require().Greater(resp.Int64(), int64(0))
}
