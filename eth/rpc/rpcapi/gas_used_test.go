package rpcapi_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	nibidcmd "github.com/NibiruChain/nibiru/v2/cmd/nibid/impl"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

func (s *BackendSuite) assertBlockGasCoversReceipts(receipts ...*rpcapi.TransactionReceipt) {
	gasUsedByBlock := make(map[int64]uint64, len(receipts))
	for _, receipt := range receipts {
		s.Require().NotNil(receipt)
		s.Require().NotNil(receipt.BlockNumber)
		gasUsedByBlock[receipt.BlockNumber.Int64()] += receipt.GasUsed
	}

	for height, expectedGasUsed := range gasUsedByBlock {
		block, err := s.cli.EvmRpc.Eth.GetBlockByNumber(rpc.NewBlockNumber(big.NewInt(height)), false)
		s.Require().NoError(err)
		s.Require().NotNil(block)
		s.Require().NotNil(block["gasUsed"])
		s.Require().GreaterOrEqual(
			block["gasUsed"].(*hexutil.Big).ToInt().Uint64(),
			expectedGasUsed,
			"block %d gasUsed should cover the receipts included in that block",
			height,
		)
	}
}

// TestGasUsedTransfers verifies that gas used is correctly calculated for simple transfers.
// Test creates 2 eth transfer txs that are supposed to be included in the same block.
// It checks that gas used is the same for both txs and the total block gas is greater than the sum of 2 gas used.
func (s *BackendSuite) TestGasUsedTransfers() {
	// Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	// Start with new block
	s.Require().NoError(s.cli.WaitForNextBlock())
	balanceBefore := s.getUnibiBalance(s.evmSenderEthAddr)

	// Send 2 similar transfers
	randomEthAddr := evmtest.NewEthPrivAcc().EthAddr
	txHash1 := s.SendNibiViaEthTransfer(randomEthAddr, amountToSend, false)
	txHash2 := s.SendNibiViaEthTransfer(randomEthAddr, amountToSend, false)

	blockNumber1, _, receipt1, _ := WaitForReceipt(s, txHash1)
	blockNumber2, _, receipt2, _ := WaitForReceipt(s, txHash2)

	s.Require().NotNil(receipt1)
	s.Require().NotNil(receipt2)

	s.Require().Equal(gethcore.ReceiptStatusSuccessful, receipt1.Status)
	s.Require().Equal(gethcore.ReceiptStatusSuccessful, receipt2.Status)

	// Expect txs are included into one block
	s.Require().Equal(blockNumber1, blockNumber2)

	// Ensure that gas used is the same for both transactions
	s.Require().Equal(receipt1.GasUsed, receipt2.GasUsed)

	// Get block receipt and check gas used
	block, err := s.cli.EvmRpc.Eth.GetBlockByNumber(rpc.NewBlockNumber(blockNumber1), false)
	s.Require().NoError(err)
	s.Require().NotNil(block)
	s.Require().NotNil(block["gasUsed"])
	s.Require().GreaterOrEqual(block["gasUsed"].(*hexutil.Big).ToInt().Uint64(), receipt1.GasUsed+receipt2.GasUsed)

	// Balance after should be equal to balance before minus gas used and amount sent
	balanceAfter := s.getUnibiBalance(s.evmSenderEthAddr)
	s.Require().Equal(
		receipt1.GasUsed+receipt2.GasUsed+2,
		balanceBefore.Uint64()-balanceAfter.Uint64(),
	)
}

// TestGasUsedFunTokens verifies that gas used is correctly calculated for
// precompile "sendToBank" txs.
//
// Test creates 3 txs, 2 successful and one failing.
//   - Successful txs gas should be refunded and failing tx should consume 100%
//     of the gas limit.
//   - It also checks that each block's gas used covers the tx receipts included
//     in that block, even when the txs land in different blocks.
func (s *BackendSuite) TestGasUsedFunTokens() {
	// Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	// Create funtoken from erc20
	erc20Addr, err := eth.NewEIP55AddrFromStr(
		s.SuccessfulTxDeployContract().Receipt.ContractAddress.Hex(),
	)
	s.Require().NoError(err)

	balanceBefore := s.getUnibiBalance(s.evmSenderEthAddr)

	txResp, err := s.cli.ExecTxCmd(
		nibidcmd.TxCmd(),
		[]string{"evm", "create-funtoken", "--erc20=" + erc20Addr.Hex()},
	)
	s.Require().NoError(err)
	s.Require().NotNil(txResp)

	randomNibiAddress := testutil.AccAddress()
	packedArgsPass, err := embeds.SmartContract_FunToken.ABI.Pack(
		"sendToBank",
		erc20Addr.Address,
		big.NewInt(1),
		randomNibiAddress.String(),
	)
	s.Require().NoError(err)

	nonce := s.getCurrentNonce(s.evmSenderEthAddr)
	txHash1 := SendTransaction(
		s,
		&gethcore.LegacyTx{
			Nonce:    nonce,
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
			Nonce:    nonce + 1,
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
			Nonce:    nonce + 2,
			To:       &precompile.PrecompileAddr_FunToken,
			Data:     packedArgsPass,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
		},
		false,
	)
	blockNumber1, _, receipt1, err1 := WaitForReceipt(s, txHash1)
	blockNumber2, _, receipt2, err2 := WaitForReceipt(s, txHash2)
	blockNumber3, _, receipt3, err3 := WaitForReceipt(s, txHash3)
	for _, err := range []error{err1, err2, err3} {
		s.NoError(err)
	}

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

	gasUsedInTxs := receipt1.GasUsed + receipt2.GasUsed + receipt3.GasUsed
	s.assertBlockGasCoversReceipts(receipt1, receipt2, receipt3)

	// Balance after should be equal to balance before minus gas used
	balanceAfter := s.getUnibiBalance(s.evmSenderEthAddr)
	balanceChange := new(big.Int).Sub(balanceAfter, balanceBefore)
	s.Require().Negative(balanceChange.Cmp(big.NewInt(0)), "txs should lower the balance, not increase it")
	s.Require().LessOrEqualf(
		gasUsedInTxs,
		new(big.Int).Abs(balanceChange).Uint64(),
		"gasUsedInTxs %d, balanceBefore %s, balanceAfter %s",
		gasUsedInTxs, balanceBefore, balanceAfter,
	)
}

// TestMultipleMsgsTxGasUsage tests that the gas is correctly consumed per message in a single transaction.
func (s *BackendSuite) TestMultipleMsgsTxGasUsage() {
	// Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	balBefore := s.getUnibiBalance(s.evmSenderEthAddr)
	nonce := s.getCurrentNonce(s.evmSenderEthAddr)

	contractCreationGasLimit := uint64(1_500_000)
	contractCallGasLimit := uint64(100_000)

	// Create series of 3 tx messages. Expecting nonce to be incremented by 3
	erc20Addr := s.SuccessfulTxDeployContract().Receipt.ContractAddress
	creationTx := s.buildContractCreationTx(nonce, contractCreationGasLimit)
	firstTransferTx := s.buildContractCallTx(*erc20Addr, nonce+1, contractCallGasLimit)
	secondTransferTx := s.buildContractCallTx(*erc20Addr, nonce+2, contractCallGasLimit)

	// Create and broadcast SDK transaction
	for _, coreTx := range []*gethcore.Transaction{
		creationTx, firstTransferTx, secondTransferTx,
	} {
		sdkTx := s.buildSDKTxWithEVMMessages(
			coreTx,
		)
		s.broadcastSDKTx(sdkTx)
	}

	_, _, receiptContractCreation, _ := WaitForReceipt(s, creationTx.Hash())
	_, _, receiptFirstTransfer, _ := WaitForReceipt(s, firstTransferTx.Hash())
	_, _, receiptSecondTransfer, _ := WaitForReceipt(s, secondTransferTx.Hash())

	s.Require().Greater(receiptContractCreation.GasUsed, uint64(0))
	s.Require().LessOrEqual(receiptContractCreation.GasUsed, contractCreationGasLimit)

	s.Require().Greater(receiptFirstTransfer.GasUsed, uint64(0))
	s.Require().LessOrEqual(receiptFirstTransfer.GasUsed, contractCallGasLimit)

	s.Require().Greater(receiptSecondTransfer.GasUsed, uint64(0))
	s.Require().LessOrEqual(receiptSecondTransfer.GasUsed, contractCallGasLimit)

	balAfter := s.getUnibiBalance(s.evmSenderEthAddr)
	balAfterU64 := balAfter.Uint64()
	balBeforeU64 := balBefore.Uint64()
	s.Require().LessOrEqual(balAfterU64, balBeforeU64, "balance must have decreased")
	gasUsedFromAllTxs := receiptContractCreation.GasUsed + receiptFirstTransfer.GasUsed + receiptSecondTransfer.GasUsed

	// "x/evm/evmstate/msg_ethereum_tx_test.go" file.
	// Light assertion is fine here. We test EIP-3529 refudn logic thoroughly
	// inside of "x/evm/evmstate".
	s.Require().LessOrEqual(
		gasUsedFromAllTxs,
		balBefore.Uint64()-balAfter.Uint64(),
	)
}
