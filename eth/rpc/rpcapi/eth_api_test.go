package rpcapi_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	geth "github.com/ethereum/go-ethereum"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"
	"github.com/NibiruChain/nibiru/v2/gosdk"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"
)

var (
	_ suite.TearDownAllSuite = (*NodeSuite)(nil)
	_ suite.SetupAllSuite    = (*NodeSuite)(nil)
)

type NodeSuite struct {
	suite.Suite
	cfg     testnetwork.Config
	network *testnetwork.Network
	val     *testnetwork.Validator

	ethClient *ethclient.Client
	ethAPI    *rpcapi.EthAPI

	fundedAccPrivateKey *ecdsa.PrivateKey
	fundedAccEthAddr    gethcommon.Address
	fundedAccNibiAddr   sdk.AccAddress

	contractData embeds.CompiledEvmContract
}

func TestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(Suite))
	suite.Run(t, new(NodeSuite))
}

// SetupSuite runs before every test in the suite. Implements the
// "suite.SetupAllSuite" interface.
func (s *NodeSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())
	testapp.EnsureNibiruPrefix()

	genState := genesis.NewTestGenesisState(app.MakeEncodingConfig())
	homeDir := s.T().TempDir()
	s.cfg = testnetwork.BuildNetworkConfig(genState)
	network, err := testnetwork.New(s.T(), homeDir, s.cfg)
	s.Require().NoError(err)

	s.network = network
	s.val = network.Validators[0]
	s.ethClient = s.val.JSONRPCClient
	s.ethAPI = s.val.EthRPC_ETH
	s.contractData = embeds.SmartContract_TestERC20

	testAccPrivateKey, _ := crypto.GenerateKey()
	s.fundedAccPrivateKey = testAccPrivateKey
	s.fundedAccEthAddr = crypto.PubkeyToAddress(testAccPrivateKey.PublicKey)
	s.fundedAccNibiAddr = eth.EthAddrToNibiruAddr(s.fundedAccEthAddr)

	funds := sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 100_000_000)) // 10 NIBI
	s.NoError(testnetwork.FillWalletFromValidator(s.fundedAccNibiAddr, funds, s.val, eth.EthBaseDenom))
	s.NoError(s.network.WaitForNextBlock())
}

// Test_ChainID EVM method: eth_chainId
func (s *NodeSuite) Test_ChainID() {
	ethChainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	s.Equal(appconst.ETH_CHAIN_ID_DEFAULT, ethChainID.Int64())
}

// Test_BlockNumber EVM method: eth_blockNumber
func (s *NodeSuite) Test_BlockNumber() {
	networkBlockNumber, err := s.network.LatestHeight()
	s.NoError(err)

	ethBlockNumber, err := s.ethClient.BlockNumber(context.Background())
	s.NoError(err)
	s.Equal(networkBlockNumber, int64(ethBlockNumber))
}

// Test_BlockByNumber EVM method: eth_getBlockByNumber
func (s *NodeSuite) Test_BlockByNumber() {
	networkBlockNumber, err := s.network.LatestHeight()
	s.NoError(err)

	ethBlock, err := s.ethClient.BlockByNumber(context.Background(), big.NewInt(networkBlockNumber))
	s.NoError(err)
	s.NoError(ethBlock.SanityCheck())
}

// Test_BalanceAt EVM method: eth_getBalance
func (s *NodeSuite) Test_BalanceAt() {
	testAccEthAddr := gethcommon.BytesToAddress(testnetwork.NewAccount(s.network, "new-user"))

	// New user balance should be 0
	balance, err := s.ethClient.BalanceAt(context.Background(), testAccEthAddr, nil)
	s.NoError(err)
	s.NotNil(balance)
	s.Equal(int64(0), balance.Int64())

	// Funded account balance should be > 0
	balance, err = s.ethClient.BalanceAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)
	s.NotNil(balance)
	s.Greater(balance.Int64(), int64(0))
}

// Test_StorageAt EVM method: eth_getStorageAt
func (s *NodeSuite) Test_StorageAt() {
	storage, err := s.ethClient.StorageAt(
		context.Background(), s.fundedAccEthAddr, gethcommon.Hash{}, nil,
	)
	s.NoError(err)
	// TODO: add more checks
	s.NotNil(storage)
}

// Test_PendingStorageAt EVM method: eth_getStorageAt | pending
func (s *NodeSuite) Test_PendingStorageAt() {
	storage, err := s.ethClient.PendingStorageAt(
		context.Background(), s.fundedAccEthAddr, gethcommon.Hash{},
	)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(storage)
}

// Test_CodeAt EVM method: eth_getCode
func (s *NodeSuite) Test_CodeAt() {
	code, err := s.ethClient.CodeAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(code)
}

// Test_PendingCodeAt EVM method: eth_getCode
func (s *NodeSuite) Test_PendingCodeAt() {
	code, err := s.ethClient.PendingCodeAt(context.Background(), s.fundedAccEthAddr)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(code)
}

// Test_EstimateGas EVM method: eth_estimateGas
func (s *NodeSuite) Test_EstimateGas() {
	testAccEthAddr := gethcommon.BytesToAddress(testnetwork.NewAccount(s.network, "new-user"))
	gasLimit := uint64(21000)
	msg := geth.CallMsg{
		From:  s.fundedAccEthAddr,
		To:    &testAccEthAddr,
		Gas:   gasLimit,
		Value: evm.NativeToWei(big.NewInt(1)),
	}
	gasEstimated, err := s.ethClient.EstimateGas(context.Background(), msg)
	s.NoError(err)
	s.Equal(fmt.Sprintf("%d", gasLimit), fmt.Sprintf("%d", gasEstimated))

	for _, msgValue := range []*big.Int{
		big.NewInt(1),
		new(big.Int).Sub(evm.NativeToWei(big.NewInt(1)), big.NewInt(1)), // 10^12 - 1
	} {
		msg.Value = msgValue
		_, err = s.ethClient.EstimateGas(context.Background(), msg)
		s.ErrorContains(err, "StateDB: wei amount is too small")
	}
}

// Test_SuggestGasPrice EVM method: eth_gasPrice
func (s *NodeSuite) Test_SuggestGasPrice() {
	// TODO: the backend method is stubbed to 0
	_, err := s.ethClient.SuggestGasPrice(context.Background())
	s.NoError(err)
}

// Test_SimpleTransferTransaction EVM method: eth_sendRawTransaction
func (s *NodeSuite) Test_SimpleTransferTransaction() {
	chainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	nonce, err := s.ethClient.PendingNonceAt(context.Background(), s.fundedAccEthAddr)
	s.NoError(err)

	recipientAddr := gethcommon.BytesToAddress(testnetwork.NewAccount(s.network, "recipient"))
	recipientBalanceBefore, err := s.ethClient.BalanceAt(context.Background(), recipientAddr, nil)
	s.Require().NoError(err)
	s.Equal(int64(0), recipientBalanceBefore.Int64())

	s.T().Log("make sure the sender has enough funds")
	weiToSend := evm.NativeToWei(big.NewInt(1))                          // 1 unibi
	funds := sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 5_000_000)) // 5 * 10^6 unibi
	s.Require().NoError(testnetwork.FillWalletFromValidator(
		s.fundedAccNibiAddr, funds, s.network.Validators[0], eth.EthBaseDenom),
	)
	s.NoError(s.network.WaitForNextBlock())

	senderBalanceBeforeWei, err := s.ethClient.BalanceAt(
		context.Background(), s.fundedAccEthAddr, nil,
	)
	s.NoError(err)

	grpcUrl := s.network.Validators[0].AppConfig.GRPC.Address
	grpcConn, err := gosdk.GetGRPCConnection(grpcUrl, true, 5)
	s.NoError(err)

	{
		querier := bank.NewQueryClient(grpcConn)
		resp, err := querier.Balance(context.Background(), &bank.QueryBalanceRequest{
			Address: s.fundedAccNibiAddr.String(),
			Denom:   eth.EthBaseDenom,
		})
		s.Require().NoError(err)
		s.Equal("105"+strings.Repeat("0", 6), resp.Balance.Amount.String())
	}

	s.T().Logf("Sending %d wei to %s", weiToSend, recipientAddr.Hex())
	signer := gethcore.LatestSignerForChainID(chainID)
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
	s.NoError(err)
	err = s.ethClient.SendTransaction(context.Background(), tx)
	s.Require().NoError(err)
	s.NoError(s.network.WaitForNextBlock())

	s.NoError(s.network.WaitForNextBlock())
	s.NoError(s.network.WaitForNextBlock())

	txReceipt, err := s.ethClient.TransactionReceipt(blankCtx, tx.Hash())
	s.NoError(err)

	s.T().Log("Assert event expectations - successful eth tx")
	{
		blockHeightOfTx := int64(txReceipt.BlockNumber.Uint64())
		blockOfTx, err := s.val.RPCClient.BlockResults(blankCtx, &blockHeightOfTx)
		s.NoError(err)
		ethTxEvents := []sdk.Event{}
		events := blockOfTx.TxsResults[0].Events
		for _, event := range events {
			if event.Type == "ethereum_tx" {
				ethTxEvents = append(ethTxEvents,
					sdk.Event{Type: event.Type, Attributes: event.Attributes},
				)
			}
		}

		eventsJson, _ := json.Marshal(events)
		s.Require().Equal(len(ethTxEvents), 2, "events: ", eventsJson)
		hash0, _ := ethTxEvents[0].GetAttribute(evm.AttributeKeyEthereumTxHash)
		hash1, _ := ethTxEvents[1].GetAttribute(evm.AttributeKeyEthereumTxHash)
		s.Require().Equal(hash0, hash1)
	}

	s.T().Log("Assert balances")
	senderBalanceAfterWei, err := s.ethClient.BalanceAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)

	costOfTx := new(big.Int).Add(
		weiToSend,
		new(big.Int).Mul((new(big.Int).SetUint64(params.TxGas)), gasPrice),
	)
	wantSenderBalWei := new(big.Int).Sub(senderBalanceBeforeWei, costOfTx)
	s.Equal(wantSenderBalWei.String(), senderBalanceAfterWei.String(), "surpising sender balance")

	recipientBalanceAfter, err := s.ethClient.BalanceAt(context.Background(), recipientAddr, nil)
	s.NoError(err)
	s.Equal(weiToSend.String(), recipientBalanceAfter.String())
}

var blankCtx = context.Background()

// Test_SmartContract includes contract deployment, query, execution
func (s *NodeSuite) Test_SmartContract() {
	chainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	nonce, err := s.ethClient.NonceAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)

	s.T().Log("Make sure the account has funds.")

	funds := sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 1_000_000_000))
	s.Require().NoError(testnetwork.FillWalletFromValidator(
		s.fundedAccNibiAddr, funds, s.network.Validators[0], eth.EthBaseDenom),
	)
	s.NoError(s.network.WaitForNextBlock())

	grpcUrl := s.network.Validators[0].AppConfig.GRPC.Address
	grpcConn, err := gosdk.GetGRPCConnection(grpcUrl, true, 5)
	s.NoError(err)

	querier := bank.NewQueryClient(grpcConn)
	resp, err := querier.Balance(context.Background(), &bank.QueryBalanceRequest{
		Address: s.fundedAccNibiAddr.String(),
		Denom:   eth.EthBaseDenom,
	})
	s.Require().NoError(err)
	// Expect 1.005 billion because of the setup function before this test.
	s.True(resp.Balance.Amount.GT(math.NewInt(1_004_900_000)), "unexpectedly low balance ", resp.Balance.Amount.String())

	s.T().Log("Deploy contract")
	signer := gethcore.LatestSignerForChainID(chainID)
	txData := s.contractData.Bytecode
	tx, err := gethcore.SignNewTx(
		s.fundedAccPrivateKey,
		signer,
		&gethcore.LegacyTx{
			Nonce: nonce,
			Gas:   100_500_000 + params.TxGasContractCreation,
			GasPrice: evm.NativeToWei(new(big.Int).Add(
				evm.BASE_FEE_MICRONIBI, big.NewInt(0),
			)),
			Data: txData,
		})
	s.Require().NoError(err)

	txBz, err := tx.MarshalBinary()
	s.NoError(err)
	txHash, err := s.ethAPI.SendRawTransaction(txBz)
	s.Require().NoError(err)

	s.T().Log("Assert: tx IS pending just after execution")
	pendingTxs, err := s.ethAPI.GetPendingTransactions()
	s.NoError(err)
	s.Require().Len(pendingTxs, 1)
	_ = s.network.WaitForNextBlock()

	s.T().Log("Assert: tx NOT pending")
	{
		wantCount := 0
		pending, err := s.ethClient.PendingTransactionCount(blankCtx)
		s.NoError(err)
		s.Require().EqualValues(uint(wantCount), pending)

		pendingTxs, err := s.ethAPI.GetPendingTransactions()
		s.NoError(err)
		s.Require().Len(pendingTxs, wantCount)

		// This query will succeed only if a receipt is found
		_, err = s.ethClient.TransactionReceipt(blankCtx, txHash)
		s.Require().Errorf(err, "receipt for txHash: %s", txHash.Hex())

		// This query succeeds if no receipt is found
		_, err = s.ethAPI.GetTransactionReceipt(txHash)
		s.Require().NoError(err)
	}

	{
		weiToSend := evm.NativeToWei(big.NewInt(1)) // 1 unibi
		s.T().Logf("Sending %d wei (sanity check)", weiToSend)
		accResp, err := s.val.EthRpcQueryClient.QueryClient.EthAccount(blankCtx,
			&evm.QueryEthAccountRequest{
				Address: s.fundedAccEthAddr.Hex(),
			})
		s.NoError(err)
		nonce := accResp.Nonce
		recipientAddr := gethcommon.BytesToAddress(testnetwork.NewAccount(s.network, "recipient"))

		signer := gethcore.LatestSignerForChainID(chainID)
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
		txHash, err := s.ethAPI.SendRawTransaction(txBz)
		s.Require().NoError(err)
		_ = s.network.WaitForNextBlock()

		txReceipt, err := s.ethClient.TransactionReceipt(blankCtx, txHash)
		s.Require().NoError(err)
		s.NotNil(txReceipt)

		txHashFromReceipt := txReceipt.TxHash
		s.Equal(txHash, txHashFromReceipt)

		// TODO: Test eth_getTransactionByHash using a JSON-RPC request at the
		// endpoint directly.
		tx, _, err = s.ethClient.TransactionByHash(blankCtx, txHash)
		s.NoError(err)
		s.NotNil(tx)
	}
}

func (s *NodeSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}
