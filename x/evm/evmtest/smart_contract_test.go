// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest_test

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *Suite) TestCreateContractTxMsg() {
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

func (s *Suite) TestCreateContractGethCoreMsg() {
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

func (s *Suite) TestExecuteContractTxMsg() {
	deps := evmtest.NewTestDeps()
	ethAcc := evmtest.NewEthAccInfo()
	contractAddress := gethcommon.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	args := evmtest.ArgsExecuteContract{
		EthAcc:          ethAcc,
		EthChainIDInt:   deps.K.EthChainID(deps.Ctx),
		GasPrice:        big.NewInt(1),
		Nonce:           deps.StateDB().GetNonce(ethAcc.EthAddr),
		ContractAddress: &contractAddress,
		Data:            nil,
	}

	ethTxMsg, err := evmtest.ExecuteContractTxMsg(args)
	s.NoError(err)
	s.Require().NoError(ethTxMsg.ValidateBasic())
}
