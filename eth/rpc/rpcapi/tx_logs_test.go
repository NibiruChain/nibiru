package rpcapi_test

import (
	"encoding/json"
	"fmt"
	"math/big"

	abci "github.com/cometbft/cometbft/abci/types"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

// TestEthLogs checks that eth txs as well as funtoken txs produce tx_logs events and update tx index properly.
// To check that, we send a series of transactions:
// - 1: simple eth transfer
// - 2: deploying erc20 contract
// - 3. creating funtoken from erc20
// - 4: creating funtoken from coin
// - 5. converting coin to erc20
// - 6. converting erc20 born token to coin via precompile
// Each tx should emit some tx logs and emit proper tx index within ethereum tx event.
func (s *BackendSuite) TestLogs() {
	// Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	// Start with fresh block
	s.network.WaitForNextBlock()

	debugLogs := make(map[string]string)

	s.T().Log("TX1: Send simple nibi transfer")
	randomEthAddr := evmtest.NewEthPrivAcc().EthAddr
	txHashFirst := s.SendNibiViaEthTransfer(randomEthAddr, amountToSend, false)
	debugLogs["s.fundedAccEthAddr"] = s.fundedAccEthAddr.Hex()
	debugLogs["addr of recipient from tx 1"] = randomEthAddr.Hex()
	debugLogs["tx hash of tx 1"] = txHashFirst.Hex()

	s.T().Log("TX2: Deploy ERC20 contract")
	_, erc20AddrTx2 := s.DeployTestContract(false)
	erc20Addr, _ := eth.NewEIP55AddrFromStr(erc20AddrTx2.String())
	debugLogs["erc20 addr deployed in tx 2"] = erc20Addr.Hex()

	var (
		txHash3 string // Tx hash hex of TX3
		txHash4 string // Tx hash hex of TX4
		txHash5 string // Tx hash hex of TX5
	)

	s.T().Log("TX3: Create FunToken from ERC20")
	nonce := s.getCurrentNonce(eth.NibiruAddrToEthAddr(s.node.Address))
	txResp, err := s.network.BroadcastMsgs(s.node.Address, &nonce, &evm.MsgCreateFunToken{
		Sender:    s.node.Address.String(),
		FromErc20: &erc20Addr,
	})
	s.Require().NoError(err)
	s.Require().NotNil(txResp)
	s.Require().Equal(
		uint32(0),
		txResp.Code,
		fmt.Sprintf("Failed to create FunToken from ERC20. RawLog: %s", txResp.RawLog),
	)
	txHash3 = txResp.TxHash

	s.T().Log("TX4: Create FunToken from unibi coin")
	nonce++
	erc20FromCoinAddr := crypto.CreateAddress(evm.EVM_MODULE_ADDRESS, s.getCurrentNonce(evm.EVM_MODULE_ADDRESS)+1)
	debugLogs["erc20FromCoinAddr (assumed funtoken address)"] = erc20FromCoinAddr.Hex()
	debugLogs["s.node.EthAddress"] = s.node.EthAddress.Hex()

	// Query the EVM params for the WNIBI address
	txResp, err = s.network.BroadcastMsgs(s.node.Address, &nonce, &evm.MsgCreateFunToken{
		Sender:        s.node.Address.String(),
		FromBankDenom: evm.EVMBankDenom,
	})
	s.Require().NoError(err)
	s.Require().NotNil(txResp)
	s.Require().Equal(
		uint32(0),
		txResp.Code,
		fmt.Sprintf("Failed to create FunToken from unibi coin. RawLog: %s", txResp.RawLog),
	)
	txHash4 = txResp.TxHash

	s.T().Log("TX5: Convert coin to EVM")
	nonce++
	txResp, err = s.network.BroadcastMsgs(s.node.Address, &nonce, &evm.MsgConvertCoinToEvm{
		Sender:   s.node.Address.String(),
		BankCoin: sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(1)),
		ToEthAddr: eth.EIP55Addr{
			Address: s.fundedAccEthAddr,
		},
	})
	s.Require().NoError(err)
	s.Require().NotNil(txResp)
	s.Require().Equal(
		uint32(0),
		txResp.Code,
		fmt.Sprintf("Failed converting coin to evm. RawLog: %s", txResp.RawLog),
	)
	txHash5 = txResp.TxHash

	s.T().Log("TX6: Send erc20 token to coin using precompile")
	randomNibiAddress := testutil.AccAddress()
	packedArgsPass, err := embeds.SmartContract_FunToken.ABI.Pack(
		"sendToBank",
		erc20Addr.Address,
		big.NewInt(1),
		randomNibiAddress.String(),
	)
	s.Require().NoError(err)
	txHashLast := SendTransaction(
		s,
		&gethcore.LegacyTx{
			Nonce:    s.getCurrentNonce(s.fundedAccEthAddr),
			To:       &precompile.PrecompileAddr_FunToken,
			Data:     packedArgsPass,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
		},
		false,
	)

	s.T().Log("Wait for all txs to be included in a block")
	blockNumFirstTx, _, _, err := WaitForReceipt(s, txHashFirst)
	s.NoError(err)
	blockNumLastTx, _, _, err := WaitForReceipt(s, txHashLast)
	s.NoError(err)
	s.Require().NotNil(blockNumFirstTx)
	s.Require().NotNil(blockNumLastTx)

	txResults := make(map[string]abci.ResponseDeliverTx)
	for idxRaw, txHash := range []string{txHash3, txHash4, txHash5} {
		idx := idxRaw + 3
		resTx, err := s.node.Querier.TxByHash(txHash)
		s.NoError(err, "expect Querier.TxByHash to succeed for tx%d", idx)
		txResJson, _ := json.MarshalIndent(resTx.TxResult, "", "  ")
		s.T().Logf("txResp for tx%d: %s", idx, txResJson)
		txResults[txHash] = resTx.TxResult
	}

	s.T().Log("parse \"eth.evm.v1.EventFunTokenCreated\" from TX3")
	tx3FunTokenCreatedEvent := new(evm.EventFunTokenCreated)
	{
		eventName := proto.MessageName(tx3FunTokenCreatedEvent)
		events := testutil.FindAbciEventsOfType(txResults[txHash3].Events, eventName)
		s.Require().Lenf(events, 1, "expect %s", eventName)

		event, _ := evm.EventFunTokenCreatedFromABCIEvent(events[0])
		tx3FunTokenCreatedEvent = event
	}
	s.Equal(
		erc20AddrTx2.Hex(),
		tx3FunTokenCreatedEvent.Erc20ContractAddress,
		"The ERC20 from TX2 and TX3 must match",
	)

	s.T().Log("parse \"eth.evm.v1.EventFunTokenCreated\" from TX4")
	tx4FunTokenCreatedEvent := new(evm.EventFunTokenCreated)
	{
		eventName := proto.MessageName(tx4FunTokenCreatedEvent)
		events := testutil.FindAbciEventsOfType(txResults[txHash4].Events, eventName)
		s.Require().Lenf(events, 1, "expect %s", eventName)

		event, _ := evm.EventFunTokenCreatedFromABCIEvent(events[0])
		tx4FunTokenCreatedEvent = event
	}

	debugLogs["evm.FEE_COLLECTOR_ADDR"] = evm.FEE_COLLECTOR_ADDR.Hex()
	debugLogs["evm.EVM_MODULE_ADDRESS"] = evm.EVM_MODULE_ADDRESS.Hex()

	debugLogsBz, _ := json.MarshalIndent(debugLogs, "", "  ")
	s.T().Logf("debugLogs: %s", debugLogsBz)

	// Check tx logs for each tx
	testCases := []TxLogsTestCase{
		{
			TxInfo:      "TX1 - simple eth transfer, should have empty logs",
			Logs:        []*gethcore.Log{},
			ExpectEthTx: true,
		},
		{
			TxInfo: "TX2 - deploying erc20 contract, should have logs",
			Logs: []*gethcore.Log{
				// minting initial balance to the account
				{
					Address: erc20AddrTx2,
					Topics: []gethcommon.Hash{
						crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
						gethcommon.Address{}.Hash(),
						s.fundedAccEthAddr.Hash(),
					},
				},
			},
			ExpectEthTx: true,
		},
		{
			TxInfo:      "TX3 - create FunToken from ERC20, no eth tx, no logs",
			Logs:        []*gethcore.Log{},
			ExpectEthTx: false,
		},
		{
			TxInfo: "TX4 - create FunToken from bank coin, no eth tx, logs for contract deployment",
			Logs: []*gethcore.Log{
				// contract ownership to evm module
				{
					Address: gethcommon.HexToAddress(
						tx4FunTokenCreatedEvent.Erc20ContractAddress),
					Topics: []gethcommon.Hash{
						crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)")),
						gethcommon.Address{}.Hash(),
						evm.EVM_MODULE_ADDRESS.Hash(),
					},
				},
			},
			ExpectEthTx: false,
		},
		{
			TxInfo: "TX5 - Convert coin to EVM, no eth tx, logs for minting tokens to the account",
			Logs: []*gethcore.Log{
				// minting to the account
				{
					Address: gethcommon.HexToAddress(
						tx4FunTokenCreatedEvent.Erc20ContractAddress),
					Topics: []gethcommon.Hash{
						crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
						gethcommon.Address{}.Hash(),
						s.fundedAccEthAddr.Hash(),
					},
				},
			},
			ExpectEthTx: false,
		},
		{
			TxInfo: "TX6 - Send erc20 token to coin using precompile, eth tx, logs for transferring tokens to evm module",
			Logs: []*gethcore.Log{
				// transfer from account to evm module
				{
					Address: erc20AddrTx2,
					Topics: []gethcommon.Hash{
						crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
						s.fundedAccEthAddr.Hash(),
						evm.EVM_MODULE_ADDRESS.Hash(),
					},
				},
			},
			ExpectEthTx: true,
		},
	}

	// Getting block results. Note: txs could be included in more than one block
	blockNumber := blockNumFirstTx.Int64()
	blockRes, err := s.backend.TendermintBlockResultByNumber(&blockNumber)
	s.Require().NoError(err)
	s.Require().NotNil(blockRes)
	txIndex := 0
	ethTxIndex := 0
	for idx, tc := range testCases {
		s.Run(tc.TxInfo, func() {
			if txIndex+1 > len(blockRes.TxsResults) {
				blockNumber++
				if blockNumber > blockNumLastTx.Int64() {
					s.Fail("TX %d not found in block results", idx)
				}
				txIndex = 0
				ethTxIndex = 0
				blockRes, err = s.backend.TendermintBlockResultByNumber(&blockNumber)
				s.Require().NoError(err)
				s.Require().NotNil(blockRes)
			}
			s.assertTxLogsAndTxIndex(
				blockRes, txIndex, ethTxIndex, tc,
			)
			txIndex++
			if tc.ExpectEthTx {
				ethTxIndex++
			}
		})
	}
}

type TxLogsTestCase struct {
	TxInfo      string // Name of the test case
	Logs        []*gethcore.Log
	ExpectEthTx bool
}

// assertTxLogsAndTxIndex gets tx results from the block and checks tx logs and tx index.
func (s *BackendSuite) assertTxLogsAndTxIndex(
	blockRes *tmrpctypes.ResultBlockResults,
	txIndex int,
	ethTxIndex int,
	tc TxLogsTestCase,
) {
	txResults := blockRes.TxsResults[txIndex]
	s.Require().Equal(uint32(0x0), txResults.Code, "tx failed, %s. RawLog: %s", tc.TxInfo, txResults.Log)

	events := blockRes.TxsResults[txIndex].Events

	foundEthTx := false
	for _, event := range events {
		if event.Type == evm.TypeUrlEventTxLog {
			eventTxLog, err := evm.EventTxLogFromABCIEvent(event)
			s.Require().NoError(err)

			logs := evm.LogsToEthereum(eventTxLog.Logs)
			if len(tc.Logs) > 0 {
				s.Require().GreaterOrEqual(len(logs), len(tc.Logs))
				s.assertTxLogsMatch(tc.Logs, logs, tc.TxInfo)
			} else {
				s.Require().NotNil(logs)
			}
		}
		if event.Type == evm.TypeUrlEventEthereumTx {
			foundEthTx = true
			if !tc.ExpectEthTx {
				s.Fail("unexpected EventEthereumTx event for non-eth tx, %s", tc.TxInfo)
			}
			eventJsonBz, err := json.MarshalIndent(event, "", "  ")
			s.NoError(err)
			ethereumTx, err := evm.EventEthereumTxFromABCIEvent(event)
			s.Require().NoErrorf(err, "event: %s", eventJsonBz)
			s.Require().Equalf(
				fmt.Sprintf("%d", ethTxIndex),
				ethereumTx.Index,
				"tx index mismatch, TxInfo: \"%s\", event: %s",
				tc.TxInfo, eventJsonBz,
			)
		}
	}
	if tc.ExpectEthTx && !foundEthTx {
		s.Fail("expected EventEthereumTx event not found, %s", tc.TxInfo)
	}
}

// assertTxLogsMatch checks that actual tx logs include the expected logs
func (s *BackendSuite) assertTxLogsMatch(
	expectedLogs []*gethcore.Log,
	actualLogs []*gethcore.Log,
	txInfo string,
) {
	actualJson, _ := json.MarshalIndent(actualLogs, "", "  ")
	expectedJson, _ := json.MarshalIndent(expectedLogs, "", "  ")

	for idx, expectedLog := range expectedLogs {
		actualLog := actualLogs[idx]
		s.Require().Equalf(
			expectedLog.Address,
			actualLog.Address,
			"log contract address mismatch, log index %d, %s\nACTUAL logs: %s\nEXPECTED logs: %s", idx, txInfo, actualJson, expectedJson,
		)

		s.Require().Equal(
			len(expectedLog.Topics),
			len(actualLog.Topics),
			fmt.Sprintf("topics length mismatch, log index %d, %s", idx, txInfo),
		)

		for idx, topic := range expectedLog.Topics {
			s.Require().Equal(
				topic,
				actualLog.Topics[idx],
				fmt.Sprintf("topic mismatch, log index %d, %s", idx, txInfo),
			)
		}
	}
}
