package backend_test

import (
	sdkmath "cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// TestNonceIncrementWithMultipleMsgsTx tests that the nonce is incremented correctly
// when multiple messages are included in a single transaction.
func (s *BackendSuite) TestNonceIncrementWithMultipleMsgsTx() {
	// Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	nonce := s.getCurrentNonce(s.fundedAccEthAddr)

	// Create series of 3 tx messages. Expecting nonce to be incremented by 3
	creationTx := s.buildContractCreationTx(nonce, 1_500_000)
	firstTransferTx := s.buildContractCallTx(testContractAddress, nonce+1, 100_000)
	secondTransferTx := s.buildContractCallTx(testContractAddress, nonce+2, 100_000)

	// Create and broadcast SDK transaction
	sdkTx := s.buildSDKTxWithEVMMessages(
		creationTx,
		firstTransferTx,
		secondTransferTx,
	)

	// Broadcast transaction
	rsp := s.broadcastSDKTx(sdkTx)
	s.Assert().NotEqual(rsp.Code, 0)
	s.Require().NoError(s.network.WaitForNextBlock())

	// Expected nonce should be incremented by 3
	currentNonce := s.getCurrentNonce(s.fundedAccEthAddr)
	s.Assert().Equal(nonce+3, currentNonce)

	// Assert all transactions included in block
	for _, tx := range []*gethcore.Transaction{creationTx, firstTransferTx, secondTransferTx} {
		blockNum, blockHash, _ := WaitForReceipt(s, tx.Hash())
		s.Require().NotNil(blockNum)
		s.Require().NotNil(blockHash)
	}
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
