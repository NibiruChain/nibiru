package rpcapi_test

import (
	"context"
	"fmt"
	nibiCommon "github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

type IntegrationSuite struct {
	suite.Suite
	cfg     testutilcli.Config
	network *testutilcli.Network

	ethClient      *ethclient.Client
	testAccEthAddr ethCommon.Address
}

func TestSuite_IntegrationSuite_RunAll(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

// SetupSuite initialize network
func (s *IntegrationSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())
	testapp.EnsureNibiruPrefix()

	genState := genesis.NewTestGenesisState(app.MakeEncodingConfig())
	homeDir := s.T().TempDir()
	s.cfg = testutilcli.BuildNetworkConfig(genState)
	network, err := testutilcli.New(s.T(), homeDir, s.cfg)
	s.Require().NoError(err)

	s.network = network
	s.ethClient = network.Validators[0].JSONRPCClient
	s.testAccEthAddr = testutilcli.NewEthAccount(s.network, "ethuser")
}

// Test_ChainID EVM method: eth_chainId
func (s *IntegrationSuite) Test_ChainID() {
	/**
	Test suite chain ID looks like: chain_12345-1
	12345 is an EVM chain ID
	*/
	ethChainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	s.Contains(s.cfg.ChainID, fmt.Sprintf("_%s-", ethChainID))
}

// Test_BlockNumber EVM method: eth_blockNumber
func (s *IntegrationSuite) Test_BlockNumber() {
	networkBlockNumber, err := s.network.LatestHeight()
	s.NoError(err)

	ethBlockNumber, err := s.ethClient.BlockNumber(context.Background())
	s.NoError(err)
	s.Equal(networkBlockNumber, int64(ethBlockNumber))
}

// Test_BlockByNumber EVM method: eth_getBlockByNumber
func (s *IntegrationSuite) Test_BlockByNumber() {
	networkBlockNumber, err := s.network.LatestHeight()
	s.NoError(err)

	ethBlock, err := s.ethClient.BlockByNumber(context.Background(), big.NewInt(networkBlockNumber))
	s.NoError(err)

	// TODO: add more checks about the eth block
	s.NotNil(ethBlock)
}

// Test_BalanceAt EVM method: eth_getBalance
func (s *IntegrationSuite) Test_BalanceAt() {
	val := s.network.Validators[0]

	testAcc := testutilcli.NewAccount(s.network, "ethuser")
	testAccEthAddr := ethCommon.BytesToAddress(testAcc.Bytes())

	balance, err := s.ethClient.BalanceAt(context.Background(), testAccEthAddr, nil)
	s.NoError(err)
	s.NotNil(balance)
	s.Equal(int64(0), balance.Int64())

	// Fund the account
	expectedBalance := 123 * nibiCommon.TO_MICRO
	funds := sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, expectedBalance))
	s.NoError(testutilcli.FillWalletFromValidator(testAcc, funds, val, denoms.NIBI))
	s.NoError(s.network.WaitForNextBlock())

	// Balance in the current block should be non 0
	balance, err = s.ethClient.BalanceAt(context.Background(), testAccEthAddr, nil)
	s.NoError(err)
	s.NotNil(balance)
	s.Equal(expectedBalance, balance.Int64())

	// Balance in the previous block should be 0
	networkBlockNumber, err := s.network.LatestHeight()
	s.NoError(err)
	balance, err = s.ethClient.BalanceAt(
		context.Background(), testAccEthAddr, big.NewInt(networkBlockNumber-2),
	)
	s.NoError(err)
	s.Equal(int64(0), balance.Int64())
}

// Test_StorageAt EVM method: eth_getStorageAt
func (s *IntegrationSuite) Test_StorageAt() {
	storage, err := s.ethClient.StorageAt(
		context.Background(), s.testAccEthAddr, ethCommon.Hash{}, nil,
	)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(storage)
}

// Test_PendingStorageAt EVM method: eth_getStorageAt | pending
func (s *IntegrationSuite) Test_PendingStorageAt() {
	storage, err := s.ethClient.PendingStorageAt(
		context.Background(), s.testAccEthAddr, ethCommon.Hash{},
	)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(storage)
}

// Test_CodeAt EVM method: eth_getCode
func (s *IntegrationSuite) Test_CodeAt() {
	code, err := s.ethClient.CodeAt(context.Background(), s.testAccEthAddr, nil)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(code)
}

// Test_PendingCodeAt EVM method: eth_getCode
func (s *IntegrationSuite) Test_PendingCodeAt() {
	code, err := s.ethClient.PendingCodeAt(context.Background(), s.testAccEthAddr)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(code)
}

// Test_EstimateGas EVM method: eth_estimateGas
func (s *IntegrationSuite) Test_EstimateGas() {
	msg := ethereum.CallMsg{
		From:  s.testAccEthAddr,
		To:    &ethCommon.Address{},
		Gas:   21000,
		Value: big.NewInt(1),
	}
	gas, err := s.ethClient.EstimateGas(context.Background(), msg)
	s.NoError(err)
	s.Greater(gas, uint64(0))
}

// Test_SuggestGasPrice EVM method: eth_gasPrice
func (s *IntegrationSuite) Test_SuggestGasPrice() {
	gas, err := s.ethClient.SuggestGasPrice(context.Background())
	s.NoError(err)
	s.Greater(gas.Int64(), int64(0))
}

// Test_SuggestGasTipCap EVM method: eth_maxPriorityFeePerGas
func (s *IntegrationSuite) Test_SuggestGasTipCap() {
	gas, err := s.ethClient.SuggestGasTipCap(context.Background())
	s.NoError(err)
	s.Greater(gas.Int64(), int64(0))
}

// Test_SendTransaction EVM method: eth_sendRawTransaction
func (s *IntegrationSuite) Test_SendTransaction() {
	chainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	nonce, err := s.ethClient.PendingNonceAt(context.Background(), s.testAccEthAddr)
	s.NoError(err)

	// Create ETH signer
	testAccPrivateKey, _ := crypto.GenerateKey()
	testAccAddr := crypto.PubkeyToAddress(testAccPrivateKey.PublicKey)
	testNibiAddr := evmtest.EthAddrToNibiruAddr(testAccAddr)

	val := s.network.Validators[0]

	// TODO: here is the problem: ETH is considering balance as atto when it should be micro
	funds := sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 23_000_000_000_000))
	s.NoError(testutilcli.FillWalletFromValidator(testNibiAddr, funds, val, denoms.NIBI))
	s.NoError(s.network.WaitForNextBlock())

	signer := types.LatestSignerForChainID(chainID)
	tx, err := types.SignNewTx(
		testAccPrivateKey,
		signer,
		&types.LegacyTx{
			Nonce:    nonce,
			To:       &ethCommon.Address{2},
			Value:    big.NewInt(1),
			Gas:      22000,
			GasPrice: big.NewInt(params.InitialBaseFee),
		})
	s.NoError(err)
	err = s.ethClient.SendTransaction(context.Background(), tx)
	s.NoError(err)
}

func (s *IntegrationSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}
