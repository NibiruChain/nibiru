package rpcapi_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	nibidcmd "github.com/NibiruChain/nibiru/v2/cmd/nibid/impl"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

// TestLogs checks that eth txs as well as funtoken txs produce tx_logs events and update tx index properly.
// To check that, we send a series of transactions:
// - 1: simple eth transfer
// - 2: deploying erc20 contract
// - 3. creating funtoken from erc20
// - 4: ensuring or creating funtoken from coin
// - 5. converting coin to erc20
// - 6. converting erc20 born token to coin via precompile
// Each tx should emit some tx logs and emit proper tx index within ethereum tx event.
func (s *BackendSuite) TestLogs() {
	// Test is broadcasting txs. Lock to avoid nonce conflicts.
	testMutex.Lock()
	defer testMutex.Unlock()

	// Start with fresh block
	s.Require().NoError(s.cli.WaitForNextBlock())

	debugLogs := make(map[string]string)

	s.T().Log("TX1: Send simple nibi transfer")
	randomEthAddr := evmtest.NewEthPrivAcc().EthAddr
	txHashFirst := s.SendNibiViaEthTransfer(randomEthAddr, amountToSend, false)
	debugLogs["s.evmSenderEthAddr"] = s.evmSenderEthAddr.Hex()
	debugLogs["addr of recipient from tx 1"] = randomEthAddr.Hex()
	debugLogs["tx hash of tx 1"] = txHashFirst.Hex()

	s.T().Log("TX2: Deploy ERC20 contract")
	txHashDeploy, erc20AddrTx2 := s.DeployTestContract(false)
	erc20Addr, _ := eth.NewEIP55AddrFromStr(erc20AddrTx2.String())
	debugLogs["erc20 addr deployed in tx 2"] = erc20Addr.Hex()

	var (
		txHash3             string // Tx hash hex of TX3
		txHash4             string // Tx hash hex of TX4, if broadcast
		txHash5             string // Tx hash hex of TX5
		txResults           = make(map[string]*sdk.TxResponse)
		tx4FunTokenCreated  *evm.EventFunTokenCreated
		nativeFunTokenERC20 gethcommon.Address
	)

	s.T().Log("TX3: Create FunToken from ERC20")
	txResp, err := s.cli.ExecTxCmd(
		nibidcmd.TxCmd(),
		[]string{"evm", "create-funtoken", "--erc20=" + erc20Addr.Hex()},
	)
	s.Require().NoError(err)
	s.Require().NotNil(txResp)
	s.Require().Equal(
		uint32(0),
		txResp.Code,
		fmt.Sprintf("Failed to create FunToken from ERC20. RawLog: %s", txResp.RawLog),
	)
	txHash3 = txResp.TxHash
	txResults[txHash3] = txResp

	s.T().Log("TX4: Ensure FunToken from bank coin")
	debugLogs["localnet validator"] = s.cli.FromAddr.String()

	// Localnet genesis injects WNIBI.sol at the expected address. The test
	// ensures the native bank-denom FunToken mapping once, then reuses it on
	// persistent localnet reruns.
	nativeFunTokenMapping, mappingExists := s.queryFunTokenMapping(evm.EVMBankDenom)
	if mappingExists {
		nativeFunTokenERC20 = nativeFunTokenMapping.Erc20Addr.Address
		debugLogs["native funtoken mapping"] = nativeFunTokenERC20.Hex()
	} else {
		erc20FromCoinAddr := crypto.CreateAddress(evm.EVM_MODULE_ADDRESS, s.getCurrentNonce(evm.EVM_MODULE_ADDRESS)+1)
		debugLogs["erc20FromCoinAddr (assumed funtoken address)"] = erc20FromCoinAddr.Hex()

		txResp, err = s.cli.ExecTxCmd(
			nibidcmd.TxCmd(),
			[]string{"evm", "create-funtoken", "--bank-denom=" + evm.EVMBankDenom},
		)
		s.Require().NoError(err)
		s.Require().NotNil(txResp)
		s.Require().Equal(
			uint32(0),
			txResp.Code,
			fmt.Sprintf("Failed to create FunToken from bank coin. RawLog: %s", txResp.RawLog),
		)
		txHash4 = txResp.TxHash
		txResults[txHash4] = txResp

		s.T().Log("parse \"eth.evm.v1.EventFunTokenCreated\" from TX4")
		tx4FunTokenCreated = s.funTokenCreatedEvent(txResp)
		nativeFunTokenERC20 = gethcommon.HexToAddress(tx4FunTokenCreated.Erc20ContractAddress)
		debugLogs["native funtoken mapping created"] = nativeFunTokenERC20.Hex()
	}

	s.T().Log("TX5: Convert coin to EVM")
	txResp, err = s.cli.ExecTxCmd(
		nibidcmd.TxCmd(),
		[]string{
			"evm",
			"convert-coin-to-evm",
			s.evmSenderEthAddr.Hex(),
			"1" + evm.EVMBankDenom,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(txResp)
	s.Require().Equal(
		uint32(0),
		txResp.Code,
		fmt.Sprintf("Failed converting coin to evm. RawLog: %s", txResp.RawLog),
	)
	txHash5 = txResp.TxHash
	txResults[txHash5] = txResp

	s.T().Log("TX6: Send erc20 token to coin using precompile")
	randomNibiAddress := testutil.NewAccAddress()
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
			Nonce:    s.getCurrentNonce(s.evmSenderEthAddr),
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

	txHashesToLog := []struct {
		idx    int
		txHash string
	}{
		{idx: 3, txHash: txHash3},
	}
	if txHash4 != "" {
		txHashesToLog = append(txHashesToLog, struct {
			idx    int
			txHash string
		}{idx: 4, txHash: txHash4})
	}
	txHashesToLog = append(txHashesToLog, struct {
		idx    int
		txHash string
	}{idx: 5, txHash: txHash5})
	for _, txHashInfo := range txHashesToLog {
		idx := txHashInfo.idx
		txHash := txHashInfo.txHash
		txResp := txResults[txHash]
		s.Require().NotNil(txResp, "expect tx response for tx%d", idx)
		s.T().Logf("txResp for tx%d: hash=%s height=%d code=%d", idx, txResp.TxHash, txResp.Height, txResp.Code)
	}

	s.T().Log("parse \"eth.evm.v1.EventFunTokenCreated\" from TX3")
	tx3FunTokenCreatedEvent := s.funTokenCreatedEvent(txResults[txHash3])
	s.Equal(
		erc20AddrTx2.Hex(),
		tx3FunTokenCreatedEvent.Erc20ContractAddress,
		"The ERC20 from TX2 and TX3 must match",
	)

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
			EthTxHash:   txHashFirst,
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
						s.evmSenderEthAddr.Hash(),
					},
				},
			},
			ExpectEthTx: true,
			EthTxHash:   txHashDeploy,
		},
		{
			TxInfo:         "TX3 - create FunToken from ERC20, no eth tx, no logs",
			Logs:           []*gethcore.Log{},
			ExpectEthTx:    false,
			CosmosTxHash:   txHash3,
			CosmosTxHeight: txResults[txHash3].Height,
		},
		{
			TxInfo: "TX5 - Convert coin to EVM, no eth tx, logs for minting tokens to the account",
			Logs: []*gethcore.Log{
				// minting to the account
				{
					Address: nativeFunTokenERC20,
					Topics: []gethcommon.Hash{
						crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
						gethcommon.Address{}.Hash(),
						s.evmSenderEthAddr.Hash(),
					},
				},
			},
			ExpectEthTx:    false,
			CosmosTxHash:   txHash5,
			CosmosTxHeight: txResults[txHash5].Height,
		},
		{
			TxInfo: "TX6 - Send erc20 token to coin using precompile, eth tx, logs for transferring tokens to evm module",
			Logs: []*gethcore.Log{
				// transfer from account to evm module
				{
					Address: erc20AddrTx2,
					Topics: []gethcommon.Hash{
						crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
						s.evmSenderEthAddr.Hash(),
						evm.EVM_MODULE_ADDRESS.Hash(),
					},
				},
			},
			ExpectEthTx: true,
			EthTxHash:   txHashLast,
		},
	}
	if tx4FunTokenCreated != nil {
		tx4TestCase := TxLogsTestCase{
			TxInfo: "TX4 - create FunToken from bank coin, no eth tx, logs for contract deployment",
			Logs: []*gethcore.Log{
				// contract ownership to evm module
				{
					Address: nativeFunTokenERC20,
					Topics: []gethcommon.Hash{
						crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)")),
						gethcommon.Address{}.Hash(),
						evm.EVM_MODULE_ADDRESS.Hash(),
					},
				},
			},
			ExpectEthTx:    false,
			CosmosTxHash:   txHash4,
			CosmosTxHeight: txResults[txHash4].Height,
		}
		testCases = append(testCases[:3], append([]TxLogsTestCase{tx4TestCase}, testCases[3:]...)...)
	}

	for _, tc := range testCases {
		s.Run(tc.TxInfo, func() {
			blockRes, txIndex, ethTxIndex := s.findTxLogTestCase(
				blockNumFirstTx.Int64(), blockNumLastTx.Int64(), tc,
			)
			s.assertTxLogsAndTxIndex(
				blockRes, txIndex, ethTxIndex, tc,
			)
		})
	}
}

func (s *BackendSuite) queryFunTokenMapping(bankDenom string) (*evm.FunToken, bool) {
	queryResp := new(evm.QueryFunTokenMappingResponse)
	err := s.cli.ExecQueryCmd(
		nibidcmd.QueryCmd(),
		[]string{"evm", "funtoken", bankDenom},
		queryResp,
	)
	if err != nil {
		if strings.Contains(err.Error(), "token mapping not found") {
			return nil, false
		}
		s.Require().NoError(err)
	}
	if queryResp.FunToken == nil {
		return nil, false
	}
	return queryResp.FunToken, true
}

func (s *BackendSuite) funTokenCreatedEvent(txResp *sdk.TxResponse) *evm.EventFunTokenCreated {
	eventName := proto.MessageName(new(evm.EventFunTokenCreated))
	events := testutil.FindAbciEventsOfType(txResp.Events, eventName)
	s.Require().Lenf(events, 1, "expect %s", eventName)

	event, err := evm.EventFunTokenCreatedFromABCIEvent(events[0])
	s.Require().NoError(err)
	return event
}

type TxLogsTestCase struct {
	TxInfo         string // Name of the test case
	Logs           []*gethcore.Log
	ExpectEthTx    bool
	EthTxHash      gethcommon.Hash
	CosmosTxHash   string
	CosmosTxHeight int64
}

func (s *BackendSuite) findTxLogTestCase(
	firstEthTxBlock int64,
	lastEthTxBlock int64,
	tc TxLogsTestCase,
) (
	blockRes *tmrpctypes.ResultBlockResults,
	txIndex int,
	ethTxIndex int,
) {
	if tc.CosmosTxHash != "" {
		return s.findCosmosTxResult(tc.CosmosTxHeight, tc.CosmosTxHash)
	}
	return s.findEthTxResult(firstEthTxBlock, lastEthTxBlock, tc.EthTxHash)
}

func (s *BackendSuite) findCosmosTxResult(
	blockHeight int64,
	txHash string,
) (
	blockRes *tmrpctypes.ResultBlockResults,
	txIndex int,
	ethTxIndex int,
) {
	blockRes, err := s.backend.TendermintBlockResultByNumber(&blockHeight)
	s.Require().NoError(err)
	s.Require().NotNil(blockRes)

	block, err := s.backend.TendermintBlockByNumber(rpc.BlockNumber(blockHeight))
	s.Require().NoError(err)
	s.Require().NotNil(block)
	s.Require().Len(block.Block.Txs, len(blockRes.TxsResults))

	for txIndex, tx := range block.Block.Txs {
		if strings.EqualFold(fmt.Sprintf("%X", tx.Hash()), strings.TrimPrefix(txHash, "0x")) {
			return blockRes, txIndex, 0
		}
	}
	s.Require().FailNowf("tx not found", "cosmos tx %s not found at height %d", txHash, blockHeight)
	return nil, 0, 0
}

func (s *BackendSuite) findEthTxResult(
	firstBlock int64,
	lastBlock int64,
	ethTxHash gethcommon.Hash,
) (
	blockRes *tmrpctypes.ResultBlockResults,
	txIndex int,
	ethTxIndex int,
) {
	for blockHeight := firstBlock; blockHeight <= lastBlock; blockHeight++ {
		blockRes, err := s.backend.TendermintBlockResultByNumber(&blockHeight)
		s.Require().NoError(err)
		s.Require().NotNil(blockRes)

		ethTxIndex = 0
		for txIndex, txResult := range blockRes.TxsResults {
			for _, event := range txResult.Events {
				if event.Type != evm.TypeUrlEventEthereumTx {
					continue
				}
				ethereumTx, err := evm.EventEthereumTxFromABCIEvent(event)
				s.Require().NoError(err)
				if strings.EqualFold(ethereumTx.EthHash, ethTxHash.Hex()) {
					return blockRes, txIndex, ethTxIndex
				}
				ethTxIndex++
			}
		}
	}
	s.Require().FailNowf("tx not found", "eth tx %s not found from height %d to %d", ethTxHash.Hex(), firstBlock, lastBlock)
	return nil, 0, 0
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
				if len(logs) < len(tc.Logs) {
					return
				}
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
