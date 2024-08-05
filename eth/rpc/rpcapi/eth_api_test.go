package rpcapi_test

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	geth "github.com/ethereum/go-ethereum"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/app/appconst"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/gosdk"

	nibirucommon "github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/common/testutil/testnetwork"
)

var (
	_ suite.TearDownAllSuite = (*TestSuite)(nil)
	_ suite.SetupAllSuite    = (*TestSuite)(nil)
)

type TestSuite struct {
	suite.Suite
	cfg     testnetwork.Config
	network *testnetwork.Network

	ethClient *ethclient.Client

	fundedAccPrivateKey *ecdsa.PrivateKey
	fundedAccEthAddr    gethcommon.Address
	fundedAccNibiAddr   sdk.AccAddress

	contractData embeds.CompiledEvmContract
}

func TestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// SetupSuite runs before every test in the suite. Implements the
// "suite.SetupAllSuite" interface.
func (s *TestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())
	testapp.EnsureNibiruPrefix()

	genState := genesis.NewTestGenesisState(app.MakeEncodingConfig())
	homeDir := s.T().TempDir()
	s.cfg = testnetwork.BuildNetworkConfig(genState)
	network, err := testnetwork.New(s.T(), homeDir, s.cfg)
	s.Require().NoError(err)

	s.network = network
	s.ethClient = network.Validators[0].JSONRPCClient

	s.contractData, err = embeds.SmartContract_TestERC20.Load()
	s.Require().NoError(err)

	testAccPrivateKey, _ := crypto.GenerateKey()
	s.fundedAccPrivateKey = testAccPrivateKey
	s.fundedAccEthAddr = crypto.PubkeyToAddress(testAccPrivateKey.PublicKey)
	s.fundedAccNibiAddr = eth.EthAddrToNibiruAddr(s.fundedAccEthAddr)

	val := s.network.Validators[0]

	funds := sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 100_000_000)) // 10 NIBI
	s.NoError(testnetwork.FillWalletFromValidator(s.fundedAccNibiAddr, funds, val, eth.EthBaseDenom))
	s.NoError(s.network.WaitForNextBlock())
}

// Test_ChainID EVM method: eth_chainId
func (s *TestSuite) Test_ChainID() {
	ethChainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	s.Equal(appconst.ETH_CHAIN_ID_DEFAULT, ethChainID.Int64())
}

// Test_BlockNumber EVM method: eth_blockNumber
func (s *TestSuite) Test_BlockNumber() {
	networkBlockNumber, err := s.network.LatestHeight()
	s.NoError(err)

	ethBlockNumber, err := s.ethClient.BlockNumber(context.Background())
	s.NoError(err)
	s.Equal(networkBlockNumber, int64(ethBlockNumber))
}

// Test_BlockByNumber EVM method: eth_getBlockByNumber
func (s *TestSuite) Test_BlockByNumber() {
	networkBlockNumber, err := s.network.LatestHeight()
	s.NoError(err)

	ethBlock, err := s.ethClient.BlockByNumber(context.Background(), big.NewInt(networkBlockNumber))
	s.NoError(err)

	// TODO: add more checks about the eth block
	s.NotNil(ethBlock)
}

// Test_BalanceAt EVM method: eth_getBalance
func (s *TestSuite) Test_BalanceAt() {
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
func (s *TestSuite) Test_StorageAt() {
	storage, err := s.ethClient.StorageAt(
		context.Background(), s.fundedAccEthAddr, gethcommon.Hash{}, nil,
	)
	s.NoError(err)
	// TODO: add more checks
	s.NotNil(storage)
}

// Test_PendingStorageAt EVM method: eth_getStorageAt | pending
func (s *TestSuite) Test_PendingStorageAt() {
	storage, err := s.ethClient.PendingStorageAt(
		context.Background(), s.fundedAccEthAddr, gethcommon.Hash{},
	)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(storage)
}

// Test_CodeAt EVM method: eth_getCode
func (s *TestSuite) Test_CodeAt() {
	code, err := s.ethClient.CodeAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(code)
}

// Test_PendingCodeAt EVM method: eth_getCode
func (s *TestSuite) Test_PendingCodeAt() {
	code, err := s.ethClient.PendingCodeAt(context.Background(), s.fundedAccEthAddr)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(code)
}

// Test_EstimateGas EVM method: eth_estimateGas
func (s *TestSuite) Test_EstimateGas() {
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
func (s *TestSuite) Test_SuggestGasPrice() {
	// TODO: the backend method is stubbed to 0
	_, err := s.ethClient.SuggestGasPrice(context.Background())
	s.NoError(err)
}

// Test_SimpleTransferTransaction EVM method: eth_sendRawTransaction
func (s *TestSuite) Test_SimpleTransferTransaction() {
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

	querier := bank.NewQueryClient(grpcConn)
	resp, err := querier.Balance(context.Background(), &bank.QueryBalanceRequest{
		Address: s.fundedAccNibiAddr.String(),
		Denom:   eth.EthBaseDenom,
	})
	s.Require().NoError(err)
	s.Equal("105"+strings.Repeat("0", 6), resp.Balance.Amount.String())

	s.T().Logf("Sending %d wei to %s", weiToSend, recipientAddr.Hex())
	signer := gethcore.LatestSignerForChainID(chainID)
	gasPrice := big.NewInt(1)
	tx, err := gethcore.SignNewTx(
		s.fundedAccPrivateKey,
		signer,
		&gethcore.LegacyTx{
			Nonce:    nonce,
			To:       &recipientAddr,
			Value:    weiToSend,
			Gas:      params.TxGas,
			GasPrice: gasPrice,
		})
	s.NoError(err)
	err = s.ethClient.SendTransaction(context.Background(), tx)
	s.Require().NoError(err)
	s.NoError(s.network.WaitForNextBlock())

	senderAmountAfterWei, err := s.ethClient.BalanceAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)

	costOfTx := new(big.Int).Add(
		weiToSend,
		new(big.Int).Mul(evm.NativeToWei(new(big.Int).SetUint64(params.TxGas)), gasPrice),
	)
	wantSenderBalWei := new(big.Int).Sub(senderBalanceBeforeWei, costOfTx)
	s.Equal(wantSenderBalWei.String(), senderAmountAfterWei.String(), "surpising sender balance")

	recipientBalanceAfter, err := s.ethClient.BalanceAt(context.Background(), recipientAddr, nil)
	s.NoError(err)
	s.Equal(weiToSend.String(), recipientBalanceAfter.String())
}

// Test_SmartContract includes contract deployment, query, execution
func (s *TestSuite) Test_SmartContract() {
	chainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	nonce, err := s.ethClient.NonceAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)

	// Deploying contract
	signer := gethcore.LatestSignerForChainID(chainID)
	txData := s.contractData.Bytecode
	tx, err := gethcore.SignNewTx(
		s.fundedAccPrivateKey,
		signer,
		&gethcore.LegacyTx{
			Nonce:    nonce,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
			Data:     txData,
		})
	s.NoError(err)
	err = s.ethClient.SendTransaction(context.Background(), tx)
	s.NoError(err)
	s.NoError(s.network.WaitForNextBlock())
	hash := tx.Hash()
	receipt, err := s.ethClient.TransactionReceipt(context.Background(), hash)
	s.NoError(err)
	contractAddress := receipt.ContractAddress

	// Querying contract: owner's balance should be 1000_000 tokens
	ownerInitialBalance := (&big.Int{}).Mul(big.NewInt(1000_000), nibirucommon.TO_ATTO)
	s.assertERC20Balance(contractAddress, s.fundedAccEthAddr, ownerInitialBalance)

	// Querying contract: recipient balance should be 0
	recipientAddr := gethcommon.BytesToAddress(testnetwork.NewAccount(s.network, "contract_recipient"))
	s.assertERC20Balance(contractAddress, recipientAddr, big.NewInt(0))

	// Execute contract: send 1000 anibi to recipient
	sendAmount := (&big.Int{}).Mul(big.NewInt(1000), nibirucommon.TO_ATTO)
	input, err := s.contractData.ABI.Pack("transfer", recipientAddr, sendAmount)
	s.NoError(err)
	nonce, err = s.ethClient.NonceAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)
	tx, err = gethcore.SignNewTx(
		s.fundedAccPrivateKey,
		signer,
		&gethcore.LegacyTx{
			Nonce:    nonce,
			To:       &contractAddress,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
			Data:     input,
		})
	s.NoError(err)
	err = s.ethClient.SendTransaction(context.Background(), tx)
	s.NoError(err)
	s.NoError(s.network.WaitForNextBlock())

	// Querying contract: owner's balance should be 999_000 tokens
	ownerBalance := (&big.Int{}).Mul(big.NewInt(999_000), nibirucommon.TO_ATTO)
	s.assertERC20Balance(contractAddress, s.fundedAccEthAddr, ownerBalance)

	// Querying contract: recipient balance should be 1000 tokens
	recipientBalance := (&big.Int{}).Mul(big.NewInt(1000), nibirucommon.TO_ATTO)
	s.assertERC20Balance(contractAddress, recipientAddr, recipientBalance)
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *TestSuite) assertERC20Balance(
	contractAddress gethcommon.Address,
	userAddress gethcommon.Address,
	expectedBalance *big.Int,
) {
	input, err := s.contractData.ABI.Pack("balanceOf", userAddress)
	s.NoError(err)
	msg := geth.CallMsg{
		From: s.fundedAccEthAddr,
		To:   &contractAddress,
		Data: input,
	}
	recipientBalanceBeforeBytes, err := s.ethClient.CallContract(context.Background(), msg, nil)
	s.NoError(err)
	balance := new(big.Int).SetBytes(recipientBalanceBeforeBytes)
	s.Equal(expectedBalance.String(), balance.String())
}
