package app_test

// import (
// 	"strconv"

// 	gethcore "github.com/ethereum/go-ethereum/core/types"
// 	gethparams "github.com/ethereum/go-ethereum/params"

// 	"github.com/NibiruChain/nibiru/x/evm/evmtest"
// )

// func (s *TestSuite) TestMsgEthereumTx_SimpleTransfer() {
// 	testCases := []struct {
// 		name     string
// 		scenario func()
// 	}{
// 		{
// 			name: "happy: AccessListTx",
// 			scenario: func() {
// 				deps := evmtest.NewTestDeps()
// 				ethAcc := deps.Sender

// 				s.T().Log("create eth tx msg")
// 				var innerTxData []byte = nil
// 				var accessList gethcore.AccessList = nil
// 				ethTxMsg, err := evmtest.NewEthTxMsgFromTxData(
// 					&deps,
// 					gethcore.AccessListTxType,
// 					innerTxData,
// 					deps.StateDB().GetNonce(ethAcc.EthAddr),
// 					accessList,
// 				)
// 				s.NoError(err)

// 				resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
// 				s.Require().NoError(err)

// 				gasUsed := strconv.FormatUint(resp.GasUsed, 10)
// 				wantGasUsed := strconv.FormatUint(gethparams.TxGas, 10)
// 				s.Equal(gasUsed, wantGasUsed)
// 			},
// 		},
// 		{
// 			name: "happy: LegacyTx",
// 			scenario: func() {
// 				deps := evmtest.NewTestDeps()
// 				ethAcc := deps.Sender

// 				s.T().Log("create eth tx msg")
// 				var innerTxData []byte = nil
// 				var accessList gethcore.AccessList = nil
// 				ethTxMsg, err := evmtest.NewEthTxMsgFromTxData(
// 					&deps,
// 					gethcore.LegacyTxType,
// 					innerTxData,
// 					deps.StateDB().GetNonce(ethAcc.EthAddr),
// 					accessList,
// 				)
// 				s.NoError(err)

// 				resp, err := deps.Chain.EvmKeeper.EthereumTx(deps.GoCtx(), ethTxMsg)
// 				s.Require().NoError(err)

// 				gasUsed := strconv.FormatUint(resp.GasUsed, 10)
// 				wantGasUsed := strconv.FormatUint(gethparams.TxGas, 10)
// 				s.Equal(gasUsed, wantGasUsed)
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		s.Run(tc.name, tc.scenario)
// 	}
// }
