package backend_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

// TestGasUsedTransfers verifies that gas used is correctly calculated for simple transfers.
// Test creates 2 eth transfer txs that are supposed to be included in the same block.
// It checks that gas used is the same for both txs and the total block gas is greater than the sum of 2 gas used.
func (s *BackendSuite) TestGasUsedTransfers() {
	// Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	// Start with new block
	s.Require().NoError(s.network.WaitForNextBlock())
	balanceBefore := s.getUnibiBalance(s.fundedAccEthAddr)

	// Send 2 similar transfers
	randomEthAddr := evmtest.NewEthPrivAcc().EthAddr
	txHash1 := s.SendNibiViaEthTransfer(randomEthAddr, amountToSend, false)
	txHash2 := s.SendNibiViaEthTransfer(randomEthAddr, amountToSend, false)

	blockNumber1, _, receipt1 := WaitForReceipt(s, txHash1)
	blockNumber2, _, receipt2 := WaitForReceipt(s, txHash2)

	s.Require().NotNil(receipt1)
	s.Require().NotNil(receipt2)

	s.Require().Equal(gethcore.ReceiptStatusSuccessful, receipt1.Status)
	s.Require().Equal(gethcore.ReceiptStatusSuccessful, receipt2.Status)

	// Expect txs are included into one block
	s.Require().Equal(blockNumber1, blockNumber2)

	// Ensure that gas used is the same for both transactions
	s.Require().Equal(receipt1.GasUsed, receipt2.GasUsed)

	// Get block receipt and check gas used
	block, err := s.backend.GetBlockByNumber(rpc.NewBlockNumber(blockNumber1), false)
	s.Require().NoError(err)
	s.Require().NotNil(block)
	s.Require().NotNil(block["gasUsed"])
	s.Require().GreaterOrEqual(block["gasUsed"].(*hexutil.Big).ToInt().Uint64(), receipt1.GasUsed+receipt2.GasUsed)

	// Balance after should be equal to balance before minus gas used and amount sent
	balanceAfter := s.getUnibiBalance(s.fundedAccEthAddr)
	s.Require().Equal(
		receipt1.GasUsed+receipt2.GasUsed+2,
		balanceBefore.Uint64()-balanceAfter.Uint64(),
	)
}

// TestGasUsedFunTokens verifies that gas used is correctly calculated for precompile "sendToBank" txs.
// Test creates 3 txs: 2 successful and one failing.
// Successful txs gas should be refunded and failing tx should consume 100% of the gas limit.
// It also checks that txs are included in the same block and block gas is greater or equals
// to the total gas used by txs.
func (s *BackendSuite) TestGasUsedFunTokens() {
	// Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	// Create funtoken from erc20
	erc20Addr, err := eth.NewEIP55AddrFromStr(testContractAddress.String())
	s.Require().NoError(err)

	_, err = s.backend.GetTransactionCount(s.fundedAccEthAddr, rpc.EthPendingBlockNumber)
	s.Require().NoError(err)

	txResp, err := s.network.BroadcastMsgs(s.node.Address, &evm.MsgCreateFunToken{
		Sender:    s.node.Address.String(),
		FromErc20: &erc20Addr,
	})
	s.Require().NoError(err)
	s.Require().NotNil(txResp)
	s.Require().NoError(s.network.WaitForNextBlock())

	randomNibiAddress := testutil.AccAddress()
	packedArgsPass, err := embeds.SmartContract_FunToken.ABI.Pack(
		"sendToBank",
		erc20Addr.Address,
		big.NewInt(1),
		randomNibiAddress.String(),
	)
	s.Require().NoError(err)

	nonce, err := s.backend.GetTransactionCount(s.fundedAccEthAddr, rpc.EthPendingBlockNumber)
	s.Require().NoError(err)

	balanceBefore := s.getUnibiBalance(s.fundedAccEthAddr)

	txHash1 := SendTransaction(
		s,
		&gethcore.LegacyTx{
			Nonce:    uint64(*nonce),
			To:       &precompile.PrecompileAddr_FunToken,
			Data:     packedArgsPass,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
		},
		false,
	)

	packedArgsFail, err := embeds.SmartContract_FunToken.ABI.Pack(
		"sendToBank",
		erc20Addr.Address,
		big.NewInt(1),
		"invalidAddress",
	)
	s.Require().NoError(err)
	txHash2 := SendTransaction( // should fail due to invalid recipient address
		s,
		&gethcore.LegacyTx{
			Nonce:    uint64(*nonce + 1),
			To:       &precompile.PrecompileAddr_FunToken,
			Data:     packedArgsFail,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
		},
		false,
	)
	txHash3 := SendTransaction(
		s,
		&gethcore.LegacyTx{
			Nonce:    uint64(*nonce + 2),
			To:       &precompile.PrecompileAddr_FunToken,
			Data:     packedArgsPass,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
		},
		false,
	)
	blockNumber1, _, receipt1 := WaitForReceipt(s, txHash1)
	blockNumber2, _, receipt2 := WaitForReceipt(s, txHash2)
	blockNumber3, _, receipt3 := WaitForReceipt(s, txHash3)

	s.Require().NotNil(receipt1)
	s.Require().NotNil(receipt2)
	s.Require().NotNil(receipt3)

	s.Require().NotNil(blockNumber1)
	s.Require().NotNil(blockNumber2)
	s.Require().NotNil(blockNumber3)

	// 1 and 3 should pass and 2 should fail
	s.Require().Equal(gethcore.ReceiptStatusSuccessful, receipt1.Status)
	s.Require().Equal(gethcore.ReceiptStatusFailed, receipt2.Status)
	s.Require().Equal(gethcore.ReceiptStatusSuccessful, receipt3.Status)

	// TX 1 and 3 should have gas used lower than specified gas limit
	s.Require().Less(receipt1.GasUsed, uint64(500_000))
	s.Require().Less(receipt3.GasUsed, uint64(500_000))

	// TX 2 should have gas used equal to specified gas limit as it failed
	s.Require().Equal(uint64(1_500_000), receipt2.GasUsed)

	block, err := s.backend.GetBlockByNumber(rpc.NewBlockNumber(blockNumber1), false)
	s.Require().NoError(err)
	s.Require().NotNil(block)
	s.Require().NotNil(block["gasUsed"])
	s.Require().GreaterOrEqual(
		block["gasUsed"].(*hexutil.Big).ToInt().Uint64(),
		receipt1.GasUsed+receipt2.GasUsed+receipt3.GasUsed,
	)

	// Balance after should be equal to balance before minus gas used
	balanceAfter := s.getUnibiBalance(s.fundedAccEthAddr)
	s.Require().Equal(
		receipt1.GasUsed+receipt2.GasUsed+receipt3.GasUsed,
		balanceBefore.Uint64()-balanceAfter.Uint64(),
	)
}

// TestMultipleMsgsTxGasUsage tests that the gas is correctly consumed per message in a single transaction.
func (s *BackendSuite) TestMultipleMsgsTxGasUsage() {
	// Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	balanceBefore := s.getUnibiBalance(s.fundedAccEthAddr)

	nonce := s.getCurrentNonce(s.fundedAccEthAddr)

	contractCreationGasLimit := uint64(1_500_000)
	contractCallGasLimit := uint64(100_000)

	// Create series of 3 tx messages. Expecting nonce to be incremented by 3
	creationTx := s.buildContractCreationTx(nonce, contractCreationGasLimit)
	firstTransferTx := s.buildContractCallTx(testContractAddress, nonce+1, contractCallGasLimit)
	secondTransferTx := s.buildContractCallTx(testContractAddress, nonce+2, contractCallGasLimit)

	// Create and broadcast SDK transaction
	sdkTx := s.buildSDKTxWithEVMMessages(
		creationTx,
		firstTransferTx,
		secondTransferTx,
	)
	s.broadcastSDKTx(sdkTx)

	_, _, receiptContractCreation := WaitForReceipt(s, creationTx.Hash())
	_, _, receiptFirstTransfer := WaitForReceipt(s, firstTransferTx.Hash())
	_, _, receiptSecondTransfer := WaitForReceipt(s, secondTransferTx.Hash())

	s.Require().Greater(receiptContractCreation.GasUsed, uint64(0))
	s.Require().LessOrEqual(receiptContractCreation.GasUsed, contractCreationGasLimit)

	s.Require().Greater(receiptFirstTransfer.GasUsed, uint64(0))
	s.Require().LessOrEqual(receiptFirstTransfer.GasUsed, contractCallGasLimit)

	s.Require().Greater(receiptSecondTransfer.GasUsed, uint64(0))
	s.Require().LessOrEqual(receiptSecondTransfer.GasUsed, contractCallGasLimit)

	balanceAfter := s.getUnibiBalance(s.fundedAccEthAddr)
	s.Require().Equal(
		receiptContractCreation.GasUsed+receiptFirstTransfer.GasUsed+receiptSecondTransfer.GasUsed,
		balanceBefore.Uint64()-balanceAfter.Uint64(),
	)
}
