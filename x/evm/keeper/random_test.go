// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

// TestRandom tests the random value generation within the EVM.
func (s *Suite) TestRandom() {
	deps := evmtest.NewTestDeps()
	deployResp, err := evmtest.DeployContract(&deps, embeds.SmartContract_TestRandom)
	s.Require().NoError(err)
	randomContractAddr := deployResp.ContractAddr

	stateDB := deps.EvmKeeper.NewStateDB(
		deps.Ctx,
		statedb.NewEmptyTxConfig(gethcommon.BytesToHash(deps.Ctx.HeaderHash())),
	)
	evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
	evmMsg := gethcore.NewMessage(
		evm.EVM_MODULE_ADDRESS, /* from */
		&randomContractAddr,    /* to */
		deps.App.EvmKeeper.GetAccNonce(deps.Ctx, evm.EVM_MODULE_ADDRESS), /* nonce */
		big.NewInt(0),             /* value */
		keeper.Erc20GasLimitQuery, /* gas limit */
		big.NewInt(0),             /* gas price */
		big.NewInt(0),             /* gas fee cap */
		big.NewInt(0),             /* gas tip cap */
		nil,                       /* data */
		gethcore.AccessList{},     /* access list */
		false,                     /* is fake */
	)
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /* tracer */, stateDB)

	// highjacked LoadERC20BigInt method as it perfectly fits the need of this test
	random1, err := deps.EvmKeeper.ERC20().LoadERC20BigInt(
		deps.Ctx, evmObj, embeds.SmartContract_TestRandom.ABI, randomContractAddr, "getRandom",
	)
	s.Require().NoError(err)
	s.Require().NotNil(random1)
	s.Require().NotZero(random1.Int64())

	// Update block time to check that random changes
	deps.Ctx = deps.Ctx.WithBlockTime(deps.Ctx.BlockTime().Add(1 * time.Second))
	random2, err := deps.EvmKeeper.ERC20().LoadERC20BigInt(
		deps.Ctx, evmObj, embeds.SmartContract_TestRandom.ABI, randomContractAddr, "getRandom",
	)
	s.Require().NoError(err)
	s.Require().NotNil(random1)
	s.Require().NotZero(random2.Int64())
	s.Require().NotEqual(random1, random2)
}
