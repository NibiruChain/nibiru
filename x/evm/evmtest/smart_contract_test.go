// Copyright (c) 2023-2024 Nibi, Inc.
package evmtest_test

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *Suite) TestCreateContractTxMsg() {
	deps := evmtest.NewTestDeps()
	ethAcc := evmtest.NewEthPrivAcc()

	args := evmtest.ArgsCreateContract{
		EthAcc:        ethAcc,
		EthChainIDInt: deps.EvmKeeper.EthChainID(deps.Ctx),
		GasPrice:      big.NewInt(1),
		Nonce:         deps.StateDB().GetNonce(ethAcc.EthAddr),
	}

	ethTxMsg, err := evmtest.CreateContractMsgEthereumTx(args)
	s.NoError(err)
	s.Require().NoError(ethTxMsg.ValidateBasic())
}

func (s *Suite) TestExecuteContractTxMsg() {
	deps := evmtest.NewTestDeps()
	ethAcc := evmtest.NewEthPrivAcc()
	contractAddress := gethcommon.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	args := evmtest.ArgsExecuteContract{
		EthAcc:          ethAcc,
		EthChainIDInt:   deps.EvmKeeper.EthChainID(deps.Ctx),
		GasPrice:        big.NewInt(1),
		Nonce:           deps.StateDB().GetNonce(ethAcc.EthAddr),
		ContractAddress: &contractAddress,
		Data:            nil,
	}

	ethTxMsg, err := evmtest.ExecuteContractTxMsg(args)
	s.NoError(err)
	s.Require().NoError(ethTxMsg.ValidateBasic())
}
