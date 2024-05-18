package keeper_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *KeeperSuite) TestMsgEthereumTx_CreateContract() {
	testCases := []struct {
		name     string
		scenario func()
	}{
		{
			name: "happy: deploy contract, sufficient gas limit",
			scenario: func() {
				deps := evmtest.NewTestDeps()
				ethAcc := deps.Sender

				s.T().Log("create eth tx msg, increase gas limit")
				gasLimit := new(big.Int).SetUint64(
					gethparams.TxGasContractCreation + 100_000,
				)
				args := evmtest.ArgsCreateContract{
					EthAcc:        ethAcc,
					EthChainIDInt: deps.K.EthChainID(deps.Ctx),
					GasPrice:      big.NewInt(1),
					Nonce:         deps.StateDB().GetNonce(ethAcc.EthAddr),
					GasLimit:      gasLimit,
				}
				ethTxMsg, err := evmtest.CreateContractTxMsg(args)
				s.NoError(err)
				s.Require().NoError(ethTxMsg.ValidateBasic())
				s.Equal(ethTxMsg.GetGas(), gasLimit.Uint64())

				resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
				s.Require().NoError(err, "resp: %s\nblock header: %s", resp, deps.Ctx.BlockHeader().ProposerAddress)
			},
		},
		{
			name: "sad: deploy contract, exceed gas limit",
			scenario: func() {
				deps := evmtest.NewTestDeps()
				ethAcc := deps.Sender

				s.T().Log("create eth tx msg, default create contract gas")
				gasLimit := gethparams.TxGasContractCreation
				args := evmtest.ArgsCreateContract{
					EthAcc:        ethAcc,
					EthChainIDInt: deps.K.EthChainID(deps.Ctx),
					GasPrice:      big.NewInt(1),
					Nonce:         deps.StateDB().GetNonce(ethAcc.EthAddr),
				}
				ethTxMsg, err := evmtest.CreateContractTxMsg(args)
				s.NoError(err)
				s.Require().NoError(ethTxMsg.ValidateBasic())
				s.Equal(ethTxMsg.GetGas(), gasLimit)

				resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
				s.Require().ErrorContains(err, core.ErrIntrinsicGas.Error(), "resp: %s\nblock header: %s", resp, deps.Ctx.BlockHeader().ProposerAddress)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, tc.scenario)
	}
}

func (s *KeeperSuite) TestMsgEthereumTx_SimpleTransfer() {
	testCases := []struct {
		name     string
		scenario func()
	}{
		{
			name: "happy: deploy contract, sufficient gas limit",
			scenario: func() {
				deps := evmtest.NewTestDeps()
				ethAcc := deps.Sender

				s.T().Log("create eth tx msg, increase gas limit")
				gasLimit := new(big.Int).SetUint64(
					gethparams.TxGasContractCreation + 100_000,
				)
				args := evmtest.ArgsCreateContract{
					EthAcc:        ethAcc,
					EthChainIDInt: deps.K.EthChainID(deps.Ctx),
					GasPrice:      big.NewInt(1),
					Nonce:         deps.StateDB().GetNonce(ethAcc.EthAddr),
					GasLimit:      gasLimit,
				}
				ethTxMsg, err := evmtest.CreateContractTxMsg(args)
				s.NoError(err)
				s.Require().NoError(ethTxMsg.ValidateBasic())
				s.Equal(ethTxMsg.GetGas(), gasLimit.Uint64())

				resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
				s.Require().NoError(err, "resp: %s\nblock header: %s", resp, deps.Ctx.BlockHeader().ProposerAddress)
			},
		},
		{
			name: "sad: deploy contract, exceed gas limit",
			scenario: func() {
				deps := evmtest.NewTestDeps()
				ethAcc := deps.Sender

				s.T().Log("create eth tx msg, default create contract gas")
				gasLimit := gethparams.TxGasContractCreation
				args := evmtest.ArgsCreateContract{
					EthAcc:        ethAcc,
					EthChainIDInt: deps.K.EthChainID(deps.Ctx),
					GasPrice:      big.NewInt(1),
					Nonce:         deps.StateDB().GetNonce(ethAcc.EthAddr),
				}
				ethTxMsg, err := evmtest.CreateContractTxMsg(args)
				s.NoError(err)
				s.Require().NoError(ethTxMsg.ValidateBasic())
				s.Equal(ethTxMsg.GetGas(), gasLimit)

				resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
				s.Require().ErrorContains(err, core.ErrIntrinsicGas.Error(), "resp: %s\nblock header: %s", resp, deps.Ctx.BlockHeader().ProposerAddress)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, tc.scenario)
	}
}
