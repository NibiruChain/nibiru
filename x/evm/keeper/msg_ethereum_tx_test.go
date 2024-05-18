package keeper_test

import (
	"math/big"

	// "github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *KeeperSuite) TestEthereumTx() {
	testCases := []struct {
		name     string
		scenario func()
	}{
		{
			name: "deploy contract",
			scenario: func() {
				deps := evmtest.NewTestDeps()
				ethAcc := deps.Sender

				s.T().Log("create eth tx msg")
				args := evmtest.ArgsCreateContract{
					EthAcc:        ethAcc,
					EthChainIDInt: deps.K.EthChainID(deps.Ctx),
					GasPrice:      big.NewInt(1),
					Nonce:         deps.StateDB().GetNonce(ethAcc.EthAddr),
				}
				ethTxMsg, err := evmtest.CreateContractTxMsg(args)
				s.NoError(err)
				s.Require().NoError(ethTxMsg.ValidateBasic())
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, tc.scenario)
	}
}
