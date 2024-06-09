// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest_test

import (
	"math/big"

	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *SuiteEVMTest) TestCreateContractTxMsg() {
	deps := evmtest.NewTestDeps()
	ethAcc := evmtest.NewEthAccInfo()

	args := evmtest.ArgsCreateContract{
		EthAcc:        ethAcc,
		EthChainIDInt: deps.K.EthChainID(deps.Ctx),
		GasPrice:      big.NewInt(1),
		Nonce:         deps.StateDB().GetNonce(ethAcc.EthAddr),
	}

	ethTxMsg, err := evmtest.CreateContractTxMsg(args)
	s.NoError(err)
	s.Require().NoError(ethTxMsg.ValidateBasic())
}

func (s *SuiteEVMTest) TestCreateContractGethCoreMsg() {
	deps := evmtest.NewTestDeps()
	ethAcc := evmtest.NewEthAccInfo()

	args := evmtest.ArgsCreateContract{
		EthAcc:        ethAcc,
		EthChainIDInt: deps.K.EthChainID(deps.Ctx),
		GasPrice:      big.NewInt(1),
		Nonce:         deps.StateDB().GetNonce(ethAcc.EthAddr),
	}

	// chain config
	cfg := evm.EthereumConfig(args.EthChainIDInt)

	// block height
	blockHeight := big.NewInt(deps.Ctx.BlockHeight())

	_, err := evmtest.CreateContractGethCoreMsg(
		args, cfg, blockHeight,
	)
	s.NoError(err)
}
