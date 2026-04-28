package rpcapi_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"

	sdkmath "cosmossdk.io/math"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"

	"github.com/stretchr/testify/suite"

	nibidcmd "github.com/NibiruChain/nibiru/v2/cmd/nibid/impl"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/localnet"
)

type Suite struct {
	suite.Suite
}

var (
	_ suite.TearDownAllSuite = (*NodeSuite)(nil)
	_ suite.SetupAllSuite    = (*NodeSuite)(nil)
)

type NodeSuite struct {
	suite.Suite

	cli            localnet.CLI
	ethAPI         *rpcapi.EthAPI
	ethQueryClient *rpc.QueryClient
	netAPI         *rpcapi.NetAPI

	fundedAccPrivateKey *ecdsa.PrivateKey
	fundedAccEthAddr    gethcommon.Address
	fundedAccNibiAddr   sdk.AccAddress

	contractData embeds.CompiledEvmContract
}

func (s *Suite) TestExpectedMethods() {
	serverCtx := server.NewDefaultContext()
	serverCtx.Logger = cmtlog.TestingLogger()
	apis := rpcapi.GetRPCAPIs(
		serverCtx, client.Context{},
		&cmtrpcclient.WSClient{},
		true, nil,
		[]string{
			rpcapi.NamespaceEth, // eth and filters services
			rpcapi.NamespaceDebug,
		},
	)
	s.Require().Len(apis, 3)
	type TestCase struct {
		ServiceName string
		Methods     []string
	}
	testCases := []TestCase{
		{
			ServiceName: "rpcapi.EthAPI",
			Methods: []string{
				"eth_accounts",
				"eth_blockNumber",
				"eth_call",
				"eth_chainId",
				"eth_estimateGas",
				"eth_feeHistory",
				"eth_fillTransaction",
				"eth_gasPrice",
				"eth_getBalance",
				"eth_getBlockByHash",
				"eth_getBlockByNumber",
				"eth_getCode",
				"eth_getPendingTransactions",
				"eth_getProof",
				"eth_getStorageAt",
				"eth_getTransactionByBlockHashAndIndex",
				"eth_getTransactionByBlockNumberAndIndex",
				"eth_getTransactionByHash",
				"eth_getTransactionCount",
				"eth_getTransactionLogs",
				"eth_getTransactionReceipt",
				"eth_maxPriorityFeePerGas",
				"eth_sendRawTransaction",
				"eth_syncing",
			},
		},
		{
			ServiceName: "rpcapi.FiltersAPI",
			Methods: []string{
				"eth_getFilterChanges",
				"eth_getFilterLogs",
				"eth_getLogs",
			},
		},
		{
			ServiceName: "rpcapi.DebugAPI",
			// See https://geth.ethereum.org/docs/interacting-with-geth/rpc/ns-debug
			Methods: []string{
				"debug_getBadBlocks",
				"debug_getRawBlock",
				"debug_getRawHeader",
				"debug_getRawReceipts",
				"debug_getRawTransaction",
				"debug_intermediateRoots",
				"debug_standardTraceBadBlockToFile",
				"debug_standardTraceBlockToFile",
				"debug_traceBadBlock",
				"debug_traceBlock",
				"debug_traceBlockByHash",
				"debug_traceBlockByNumber",
				"debug_traceBlockFromFile",
				"debug_traceCall",
				"debug_traceChain",
				"debug_traceTransaction",
			},
		},
	}

	for idx, api := range apis {
		tc := testCases[idx]
		testName := fmt.Sprintf("%v-%v", api.Namespace, tc.ServiceName)
		s.Run(testName, func() {
			gotMethods := rpcapi.ParseAPIMethods(api)
			for _, wantMethod := range tc.Methods {
				_, ok := gotMethods[wantMethod]
				if !ok {
					errMsg := fmt.Sprintf(
						"Missing RPC implementation for \"%s\" : service: %s, namespace: %s",
						wantMethod, tc.ServiceName, api.Namespace,
					)
					s.Fail(errMsg)
				}
			}

			if s.T().Failed() {
				gotNames := []string{}
				for name := range gotMethods {
					gotNames = append(gotNames, name)
				}
				sort.Strings(gotNames)
				bz, _ := json.MarshalIndent(gotNames, "", "  ")
				s.T().Logf("gotMethods: %s", bz)
			}
		})
	}
}

// SetupSuite runs before every test in the suite. Implements the
// "suite.SetupAllSuite" interface.
func (s *NodeSuite) SetupSuite() {
	if err := nutil.EnsureLocalBlockchain(); err != nil {
		s.T().Skipf("localnet unavailable: %v", err)
	}

	localnetCLI, err := localnet.NewCLI()
	s.Require().NoError(err)
	s.cli = localnetCLI

	s.ethAPI = localnetCLI.EvmRpc.Eth
	s.ethQueryClient = rpc.NewQueryClient(localnetCLI.ClientCtx)
	s.netAPI = localnetCLI.EvmRpc.Net
	s.contractData = embeds.SmartContract_TestERC20

	testAccPrivateKey, _ := crypto.GenerateKey()
	s.fundedAccPrivateKey = testAccPrivateKey
	s.fundedAccEthAddr = crypto.PubkeyToAddress(testAccPrivateKey.PublicKey)
	s.fundedAccNibiAddr = eth.EthAddrToNibiruAddr(s.fundedAccEthAddr)

	funds := sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 100_000_000)) // 10 NIBI
	txResp, err := s.cli.ExecTxCmd(
		nibidcmd.TxCmd(),
		[]string{"bank", "send", localnet.KeyName, s.fundedAccNibiAddr.String(), funds.String()},
	)
	s.Require().NoError(err)
	s.Require().NotNil(txResp.TxHash)
}

// Test_ChainID EVM method: eth_chainId
func (s *NodeSuite) Test_ChainID() {
	ethChainID, err := s.ethAPI.ChainId()
	s.NoError(err)
	s.Equal(appconst.ETH_CHAIN_ID_DEFAULT, ethChainID.ToInt().Int64())
}

// Test_BlockNumber EVM method: eth_blockNumber
func (s *NodeSuite) Test_BlockNumber() {
	networkBlockNumber, err := s.cli.LatestHeight()
	s.NoError(err)

	ethBlockNumber, err := s.ethAPI.BlockNumber()
	s.NoError(err)
	// It might be off by 1 block in either direction.
	blockDiff := networkBlockNumber - int64(ethBlockNumber)
	s.Truef(blockDiff <= 2, "networkBlockNumber %d, ethBlockNumber %d",
		networkBlockNumber, ethBlockNumber,
	)
}

// Test_BlockByNumber EVM method: eth_getBlockByNumber
func (s *NodeSuite) Test_BlockByNumber() {
	networkBlockNumber, err := s.cli.LatestHeight()
	s.NoError(err)

	ethBlock, err := s.ethAPI.GetBlockByNumber(rpc.NewBlockNumber(big.NewInt(networkBlockNumber)), true)
	s.NoError(err)
	s.NotNil(ethBlock)
	s.Equal(networkBlockNumber, int64(ethBlock["number"].(hexutil.Uint64)))
}

// Test_BalanceAt EVM method: eth_getBalance
func (s *NodeSuite) Test_BalanceAt() {
	testAccEthAddr := evmtest.NewEthPrivAcc().EthAddr

	// New user balance should be 0
	balance, err := s.ethAPI.GetBalance(testAccEthAddr, latestBlockOrHash())
	s.NoError(err)
	s.NotNil(balance)
	s.Equal(int64(0), balance.ToInt().Int64())

	// Funded account balance should be > 0
	balance, err = s.ethAPI.GetBalance(s.fundedAccEthAddr, latestBlockOrHash())
	s.NoError(err)
	s.NotNil(balance)
	s.Greater(balance.ToInt().Int64(), int64(0))
}

// Test_StorageAt EVM method: eth_getStorageAt
func (s *NodeSuite) Test_StorageAt() {
	storage, err := s.ethAPI.GetStorageAt(
		s.fundedAccEthAddr, gethcommon.Hash{}.Hex(), latestBlockOrHash(),
	)
	s.NoError(err)
	// TODO: add more checks
	s.NotNil(storage)
}

// Test_PendingStorageAt EVM method: eth_getStorageAt | pending
func (s *NodeSuite) Test_PendingStorageAt() {
	storage, err := s.ethAPI.GetStorageAt(
		s.fundedAccEthAddr, gethcommon.Hash{}.Hex(), pendingBlockOrHash(),
	)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(storage)
}

// Test_CodeAt EVM method: eth_getCode
func (s *NodeSuite) Test_CodeAt() {
	code, err := s.ethAPI.GetCode(s.fundedAccEthAddr, latestBlockOrHash())
	s.NoError(err)

	s.Empty(code)
}

// Test_PendingCodeAt EVM method: eth_getCode
func (s *NodeSuite) Test_PendingCodeAt() {
	code, err := s.ethAPI.GetCode(s.fundedAccEthAddr, pendingBlockOrHash())
	s.NoError(err)
	s.Empty(code)
}

// Test_EstimateGas EVM method: eth_estimateGas
func (s *NodeSuite) Test_EstimateGas() {
	testAccEthAddr := evmtest.NewEthPrivAcc().EthAddr
	gasLimit := uint64(21000)
	gasHex := hexutil.Uint64(gasLimit)
	msg := evm.JsonTxArgs{
		From:  &s.fundedAccEthAddr,
		To:    &testAccEthAddr,
		Gas:   &gasHex,
		Value: (*hexutil.Big)(evm.NativeToWei(big.NewInt(1))),
	}
	gasEstimated, err := s.ethAPI.EstimateGas(msg, nil)
	s.NoError(err)
	s.Equal(fmt.Sprintf("%d", gasLimit), fmt.Sprintf("%d", uint64(gasEstimated)))

	for _, msgValue := range []*big.Int{
		big.NewInt(1),
		new(big.Int).Sub(evm.NativeToWei(big.NewInt(1)), big.NewInt(1)), // 10^12 - 1
	} {
		msg.Value = (*hexutil.Big)(msgValue)
		_, err = s.ethAPI.EstimateGas(msg, nil)
		s.NoError(err, "estimate gas should work")
	}
}

// Test_SuggestGasPrice EVM method: eth_gasPrice
func (s *NodeSuite) Test_SuggestGasPrice() {
	// TODO: the backend method is stubbed to 0
	_, err := s.ethAPI.GasPrice()
	s.NoError(err)
}

// Test_SimpleTransferTransaction EVM method: eth_sendRawTransaction
func (s *NodeSuite) Test_SimpleTransferTransaction() {
	chainID, err := s.ethAPI.ChainId()
	s.NoError(err)
	nonce, err := s.ethAPI.GetTransactionCount(s.fundedAccEthAddr, pendingBlockOrHash())
	s.NoError(err)

	recipientAddr := evmtest.NewEthPrivAcc().EthAddr
	recipientBalanceBefore, err := s.ethAPI.GetBalance(recipientAddr, latestBlockOrHash())
	s.Require().NoError(err)
	s.Equal(int64(0), recipientBalanceBefore.ToInt().Int64())

	s.T().Log("make sure the sender has enough funds")
	weiToSend := evm.NativeToWei(big.NewInt(1))                          // 1 unibi
	funds := sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 5_000_000)) // 5 * 10^6 unibi
	txResp, err := s.cli.ExecTxCmd(
		nibidcmd.TxCmd(),
		[]string{"bank", "send", localnet.KeyName, s.fundedAccNibiAddr.String(), funds.String()},
	)
	s.Require().NoError(err)
	s.Require().NotNil(txResp.TxHash)

	senderBalanceBeforeWei, err := s.ethAPI.GetBalance(s.fundedAccEthAddr, latestBlockOrHash())
	s.NoError(err)

	{
		querier := bank.NewQueryClient(s.cli.ClientCtx)
		resp, err := querier.Balance(context.Background(), &bank.QueryBalanceRequest{
			Address: s.fundedAccNibiAddr.String(),
			Denom:   eth.EthBaseDenom,
		})
		s.Require().NoError(err)
		s.Equal("105"+strings.Repeat("0", 6), resp.Balance.Amount.String())
	}

	s.T().Logf("Sending %d wei to %s", weiToSend, recipientAddr.Hex())
	signer := gethcore.LatestSignerForChainID(chainID.ToInt())
	gasPrice := evm.NativeToWei(big.NewInt(1))
	tx, err := gethcore.SignNewTx(
		s.fundedAccPrivateKey,
		signer,
		&gethcore.LegacyTx{
			Nonce:    uint64(*nonce),
			To:       &recipientAddr,
			Value:    weiToSend,
			Gas:      params.TxGas,
			GasPrice: gasPrice, // 1 micronibi per gas
		})
	s.NoError(err)
	txBz, err := tx.MarshalBinary()
	s.Require().NoError(err)
	resTxHash, err := s.ethAPI.SendRawTransaction(txBz)
	s.Require().NoError(err)
	s.Require().Equal(tx.Hash(), resTxHash)
	s.Require().NoError(s.cli.WaitForNextBlock())

	txReceipt, err := s.waitForEthReceipt(tx.Hash())
	s.NoError(err)
	s.Require().Equal(tx.Hash(), txReceipt.TxHash)

	s.T().Log("Assert balances")
	senderBalanceAfterWei, err := s.ethAPI.GetBalance(s.fundedAccEthAddr, latestBlockOrHash())
	s.NoError(err)

	costOfTx := new(big.Int).Add(
		weiToSend,
		new(big.Int).Mul((new(big.Int).SetUint64(params.TxGas)), gasPrice),
	)
	wantSenderBalWei := new(big.Int).Sub(senderBalanceBeforeWei.ToInt(), costOfTx)
	s.Equal(wantSenderBalWei.String(), senderBalanceAfterWei.ToInt().String(), "surpising sender balance")

	recipientBalanceAfter, err := s.ethAPI.GetBalance(recipientAddr, latestBlockOrHash())
	s.NoError(err)
	s.Equal(weiToSend.String(), recipientBalanceAfter.ToInt().String())
}

var blankCtx = context.Background()

func latestBlockOrHash() rpc.BlockNumberOrHash {
	latest := rpc.EthLatestBlockNumber
	return rpc.BlockNumberOrHash{BlockNumber: &latest}
}

func pendingBlockOrHash() rpc.BlockNumberOrHash {
	pending := rpc.EthPendingBlockNumber
	return rpc.BlockNumberOrHash{BlockNumber: &pending}
}

func (s *NodeSuite) waitForEthReceipt(txHash gethcommon.Hash) (*rpcapi.TransactionReceipt, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		txReceipt, err := s.ethAPI.GetTransactionReceipt(txHash)
		if err == nil && txReceipt != nil {
			return txReceipt, nil
		}
		lastErr = err
		if attempt < 2 {
			s.Require().NoError(s.cli.WaitForNextBlock())
		}
	}
	return nil, fmt.Errorf("receipt not found after waiting two blocks for tx %s: %w", txHash.Hex(), lastErr)
}

// Test_SmartContract includes contract deployment, query, execution
func (s *NodeSuite) Test_SmartContract() {
	chainID, err := s.ethAPI.ChainId()
	s.NoError(err)
	nonce, err := s.ethAPI.GetTransactionCount(s.fundedAccEthAddr, latestBlockOrHash())
	s.NoError(err)

	s.T().Log("Make sure the account has funds.")

	funds := sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 1_000_000_000))
	txResp, err := s.cli.ExecTxCmd(
		nibidcmd.TxCmd(),
		[]string{"bank", "send", localnet.KeyName, s.fundedAccNibiAddr.String(), funds.String()},
	)
	s.Require().NoError(err)
	s.Require().NotNil(txResp.TxHash)

	querier := bank.NewQueryClient(s.cli.ClientCtx)
	resp, err := querier.Balance(context.Background(), &bank.QueryBalanceRequest{
		Address: s.fundedAccNibiAddr.String(),
		Denom:   eth.EthBaseDenom,
	})
	s.Require().NoError(err)
	// Expect 1.005 billion because of the setup function before this test.
	s.True(resp.Balance.Amount.GT(sdkmath.NewInt(1_004_900_000)), "unexpectedly low balance ", resp.Balance.Amount.String())

	s.T().Log("Deploy contract")
	{
		signer := gethcore.LatestSignerForChainID(chainID.ToInt())
		txData := s.contractData.Bytecode
		tx, err := gethcore.SignNewTx(
			s.fundedAccPrivateKey,
			signer,
			&gethcore.LegacyTx{
				Nonce: uint64(*nonce),
				Gas:   100_500_000 + params.TxGasContractCreation,
				GasPrice: evm.NativeToWei(new(big.Int).Add(
					evm.BASE_FEE_MICRONIBI, big.NewInt(0),
				)),
				Data: txData,
			})
		s.Require().NoError(err)

		txBz, err := tx.MarshalBinary()
		s.Require().NoError(err)
		txHash, err := s.ethAPI.SendRawTransaction(txBz)
		s.Require().NoError(err)
		s.Require().Equal(tx.Hash(), txHash)

		s.T().Log("Wait one block so the tx won't be pending")
		s.Require().NoError(s.cli.WaitForNextBlock())

		s.T().Log("Assert: tx NOT pending")

		var pendingTxs []*rpc.EthTxJsonRPC
		pendingTxs, err = s.ethAPI.GetPendingTransactions()
		s.NoError(err)
		for _, pendingTx := range pendingTxs {
			s.Require().NotEqual(txHash, pendingTx.Hash)
		}

		txReceipt, err := s.waitForEthReceipt(txHash)
		s.Require().NoErrorf(err, "receipt for txHash: %s", txHash.Hex())
		s.Equal(txHash, txReceipt.TxHash)

		rpcTx, err := s.ethAPI.GetTransactionByHash(txHash)
		s.NoError(err)
		s.Require().NotNil(rpcTx)
		s.Require().NotNil(rpcTx.BlockHash)
		s.Require().NotNil(rpcTx.BlockNumber)
	}

	{
		weiToSend := evm.NativeToWei(big.NewInt(1)) // 1 unibi
		s.T().Logf("Sending %d wei (sanity check)", weiToSend)
		accResp, err := s.ethQueryClient.EthAccount(blankCtx,
			&evm.QueryEthAccountRequest{
				Address: s.fundedAccEthAddr.Hex(),
			})
		s.NoError(err)
		nonce := accResp.Nonce
		recipientAddr := evmtest.NewEthPrivAcc().EthAddr

		signer := gethcore.LatestSignerForChainID(chainID.ToInt())
		gasPrice := evm.NativeToWei(big.NewInt(1))
		tx, err := gethcore.SignNewTx(
			s.fundedAccPrivateKey,
			signer,
			&gethcore.LegacyTx{
				Nonce:    nonce,
				To:       &recipientAddr,
				Value:    weiToSend,
				Gas:      params.TxGas,
				GasPrice: gasPrice, // 1 micronibi per gas
			})
		s.Require().NoError(err)
		txBz, err := tx.MarshalBinary()
		s.NoError(err)

		resTxHash, err := s.ethAPI.SendRawTransaction(txBz)
		s.Require().NoError(err)
		s.Require().NoError(s.cli.WaitForNextBlock())
		s.Equal(tx.Hash().Hex(), resTxHash.Hex())

		txReceipt, err := s.waitForEthReceipt(resTxHash)
		s.Require().NoError(err)
		s.NotNil(txReceipt)
		txHashFromReceipt := txReceipt.TxHash
		s.Equal(resTxHash.Hex(), txHashFromReceipt.Hex())

		txJSON, err := s.ethAPI.GetTransactionByHash(txHashFromReceipt)
		s.NoError(err)
		s.NotNil(txJSON)
		s.Equal(txHashFromReceipt, txJSON.Hash)
	}
}

func (s *NodeSuite) TearDownSuite() {
	s.Require().NoError(s.cli.Close())
	s.T().Log("NodeSuite uses persistent localnet; cleaned up in-process clients")
}
