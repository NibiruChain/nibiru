package rpcapi_test

import (
	"encoding/json"

	sdkmath "cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// TestNonceIncrementWithMultipleMsgsTx tests that the nonce is incremented correctly
// when multiple messages are included in a single transaction.
func (s *BackendSuite) TestNonceIncrementWithMultipleMsgsTx() { // Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	nonce := s.getCurrentNonce(s.fundedAccEthAddr)
	s.T().Logf("Before txs, nonce = %d", nonce)

	erc20Addr := s.SuccessfulTxDeployContract().Receipt.ContractAddress
	// Create series of 3 tx messages. Expecting nonce to be incremented by 3
	txMsgs := []struct {
		name   string
		coreTx *gethcore.Transaction
	}{
		{name: "creationTx", coreTx: s.buildContractCreationTx(nonce, 1_500_000)},
		{name: "firstTransferTx", coreTx: s.buildContractCallTx(*erc20Addr, nonce+1, 100_000)},
		{name: "secondTransferTx", coreTx: s.buildContractCallTx(*erc20Addr, nonce+2, 100_000)},
	}

	// Create and broadcast SDK transaction
	sdkTx := s.buildSDKTxWithEVMMessages(
		txMsgs[0].coreTx,
		txMsgs[1].coreTx,
		txMsgs[2].coreTx,
	)

	s.T().Log("Broadcast transaction. Expect failure in ante handler")
	rsp := s.broadcastSDKTx(sdkTx)
	{
		jsonBz, err := json.MarshalIndent(rsp, "", "  ")
		s.NoError(err)

		s.NotEqualValuesf(rsp.Code, 0, "expect tx not to be included yet. sdk.TxResp: %s", jsonBz)
		s.network.WaitForNextBlock()
		s.Require().NotEqualValuesf(rsp.Code, 0, "expect tx to fail. sdk.TxResp: %s", jsonBz)
		s.Contains(rsp.RawLog, "Ethereum transaction must be exactly one tx msg: got 3")
	}

	// TODO: UD-DEBUG: Nonce should be the same and only increse after successful
	// txs.
	currentNonce := s.getCurrentNonce(s.fundedAccEthAddr)
	s.Assert().Equal(nonce, currentNonce, "expect nonce to be the same")

	s.T().Logf("After failed txs, nonce = %d (unchanged)", nonce)

	for _, txMsg := range txMsgs {
		receipt, err := s.backend.GetTransactionReceipt(txMsg.coreTx.Hash())
		s.Nilf(receipt, "expect no receipt to be found | %v", txMsg.name)
		s.NoErrorf(err, "expect no error. Becuase the query succeeded but returns a blank receipt | %v", txMsg.name)
	}

	s.T().Log("Broadcast 3 happy txs. Expect nonce to increment by 3")
	// TODO: perf(evmante): Make per-block uncommitted txs execute in a batch,
	// handling pending txs in the nonce check.
	for _, txMsg := range txMsgs {
		sdkTx := s.buildSDKTxWithEVMMessages(txMsg.coreTx)
		rsp = s.broadcastSDKTx(sdkTx)
		s.EqualValuesf(rsp.Code, 0, "expect broadcast | %v", txMsg.name)
		s.Empty(rsp.RawLog, "expect no errors in raw log | %v", txMsg.name)

		jsonBz, err := json.MarshalIndent(rsp, "", "  ")
		s.NoError(err)
		s.T().Logf("sdk.TxResp %v: %s", txMsg.name, jsonBz)
	}

	s.network.WaitForNextBlock()

	currentNonce = s.getCurrentNonce(s.fundedAccEthAddr)
	s.Assert().Equal(nonce+3, currentNonce)

	// ## Notes from when we expected the tx to succeed.
	// This tx includes more than one EthereumTx msg, making it an invalid
	// `sdk.Tx`, because we enforce each `evm.Tx` to be only one EthereumTx.
	//
	// s.T().Log("Assert all transactions included in block")
	// for _, tx := range []*gethcore.Transaction{creationTx, firstTransferTx, secondTransferTx} {
	// 	blockNum, blockHash, _, _ := WaitForReceipt(s, tx.Hash())
	// 	s.Require().NotNil(blockNum)
	// 	s.Require().NotNil(blockHash)
	// }
}

// buildSDKTxWithEVMMessages creates an SDK transaction with EVM messages
func (s *BackendSuite) buildSDKTxWithEVMMessages(txs ...*gethcore.Transaction) sdk.Tx {
	msgs := make([]sdk.Msg, len(txs))
	for i, tx := range txs {
		msg := &evm.MsgEthereumTx{}
		err := msg.FromEthereumTx(tx)
		s.Require().NoError(err)
		msgs[i] = msg
	}

	option, err := codectypes.NewAnyWithValue(&evm.ExtensionOptionsEthereumTx{})
	s.Require().NoError(err)

	txBuilder, _ := s.backend.ClientCtx().TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
	txBuilder.SetExtensionOptions(option)
	err = txBuilder.SetMsgs(msgs...)
	s.Require().NoError(err)

	// Set fees for all messages
	totalGas := uint64(0)
	for _, tx := range txs {
		totalGas += tx.Gas()
	}
	fees := sdk.NewCoins(sdk.NewCoin("unibi", sdkmath.NewIntFromUint64(totalGas)))
	txBuilder.SetFeeAmount(fees)
	txBuilder.SetGasLimit(totalGas)

	return txBuilder.GetTx()
}
