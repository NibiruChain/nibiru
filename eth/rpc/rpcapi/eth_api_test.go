package rpcapi_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/NibiruChain/nibiru/app/appconst"
	fixture "github.com/NibiruChain/nibiru/eth/rpc/rpcapi/fixture"
	nibiCommon "github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/status-im/keycard-go/hexutils"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

type ContractJson struct {
	Bytecode string  `json:"bytecode"`
	Abi      abi.ABI `json:"abi"`
}

type IntegrationSuite struct {
	suite.Suite
	cfg     testutilcli.Config
	network *testutilcli.Network

	ethClient *ethclient.Client

	fundedAccPrivateKey *ecdsa.PrivateKey
	fundedAccEthAddr    ethCommon.Address
	fundedAccNibiAddr   sdk.AccAddress

	contractData *ContractJson
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

	s.contractData = getContractData()

	testAccPrivateKey, _ := crypto.GenerateKey()
	s.fundedAccPrivateKey = testAccPrivateKey
	s.fundedAccEthAddr = crypto.PubkeyToAddress(testAccPrivateKey.PublicKey)
	s.fundedAccNibiAddr = evmtest.EthAddrToNibiruAddr(s.fundedAccEthAddr)

	val := s.network.Validators[0]

	funds := sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 100_000_000)) // 10 NIBI
	s.NoError(testutilcli.FillWalletFromValidator(s.fundedAccNibiAddr, funds, val, denoms.NIBI))
	s.NoError(s.network.WaitForNextBlock())
}

// Test_ChainID EVM method: eth_chainId
func (s *IntegrationSuite) Test_ChainID() {
	ethChainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	s.Equal(appconst.ETH_CHAIN_ID_DEFAULT, ethChainID.Int64())
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
	testAccEthAddr := ethCommon.BytesToAddress(testutilcli.NewAccount(s.network, "new-user"))

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
func (s *IntegrationSuite) Test_StorageAt() {
	storage, err := s.ethClient.StorageAt(
		context.Background(), s.fundedAccEthAddr, ethCommon.Hash{}, nil,
	)
	s.NoError(err)
	// TODO: add more checks
	s.NotNil(storage)
}

// Test_PendingStorageAt EVM method: eth_getStorageAt | pending
func (s *IntegrationSuite) Test_PendingStorageAt() {
	storage, err := s.ethClient.PendingStorageAt(
		context.Background(), s.fundedAccEthAddr, ethCommon.Hash{},
	)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(storage)
}

// Test_CodeAt EVM method: eth_getCode
func (s *IntegrationSuite) Test_CodeAt() {
	code, err := s.ethClient.CodeAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(code)
}

// Test_PendingCodeAt EVM method: eth_getCode
func (s *IntegrationSuite) Test_PendingCodeAt() {
	code, err := s.ethClient.PendingCodeAt(context.Background(), s.fundedAccEthAddr)
	s.NoError(err)

	// TODO: add more checks
	s.NotNil(code)
}

// Test_EstimateGas EVM method: eth_estimateGas
func (s *IntegrationSuite) Test_EstimateGas() {
	testAccEthAddr := ethCommon.BytesToAddress(testutilcli.NewAccount(s.network, "new-user"))
	gasLimit := uint64(21000)
	msg := ethereum.CallMsg{
		From:  s.fundedAccEthAddr,
		To:    &testAccEthAddr,
		Gas:   gasLimit,
		Value: big.NewInt(1),
	}
	gasEstimated, err := s.ethClient.EstimateGas(context.Background(), msg)
	s.NoError(err)
	s.Equal(gasEstimated, gasLimit)
}

// Test_SuggestGasPrice EVM method: eth_gasPrice
func (s *IntegrationSuite) Test_SuggestGasPrice() {
	// TODO: the backend method is stubbed to 0
	_, err := s.ethClient.SuggestGasPrice(context.Background())
	s.NoError(err)

}

// Test_SimpleTransferTransaction EVM method: eth_sendRawTransaction
func (s *IntegrationSuite) Test_SimpleTransferTransaction() {
	chainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	nonce, err := s.ethClient.PendingNonceAt(context.Background(), s.fundedAccEthAddr)
	s.NoError(err)

	senderBalanceBefore, err := s.ethClient.BalanceAt(
		context.Background(), s.fundedAccEthAddr, nil,
	)
	recipientAddr := ethCommon.BytesToAddress(testutilcli.NewAccount(s.network, "recepient"))
	recipientBalanceBefore, err := s.ethClient.BalanceAt(context.Background(), recipientAddr, nil)
	s.NoError(err)
	s.Equal(int64(0), recipientBalanceBefore.Int64())

	amountToSend := big.NewInt(1000)

	signer := types.LatestSignerForChainID(chainID)
	tx, err := types.SignNewTx(
		s.fundedAccPrivateKey,
		signer,
		&types.LegacyTx{
			Nonce:    nonce,
			To:       &recipientAddr,
			Value:    amountToSend,
			Gas:      params.TxGas,
			GasPrice: big.NewInt(1),
		})
	s.NoError(err)
	err = s.ethClient.SendTransaction(context.Background(), tx)
	s.NoError(err)
	s.NoError(s.network.WaitForNextBlock())

	senderAmountAfter, err := s.ethClient.BalanceAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)

	expectedSenderBalance := senderBalanceBefore.Sub(senderBalanceBefore, amountToSend)
	expectedSenderBalance = expectedSenderBalance.Sub(senderBalanceBefore, big.NewInt(int64(params.TxGas)))

	s.Equal(expectedSenderBalance.Int64(), senderAmountAfter.Int64())

	recipientBalanceAfter, err := s.ethClient.BalanceAt(context.Background(), recipientAddr, nil)
	s.NoError(err)
	s.Equal(amountToSend.Int64(), recipientBalanceAfter.Int64())
}

// Test_SmartContract includes contract deployment, query, execution
func (s *IntegrationSuite) Test_SmartContract() {
	chainID, err := s.ethClient.ChainID(context.Background())
	s.NoError(err)
	nonce, err := s.ethClient.NonceAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)

	// Deploying contract
	signer := types.LatestSignerForChainID(chainID)
	tx, err := types.SignNewTx(
		s.fundedAccPrivateKey,
		signer,
		&types.LegacyTx{
			Nonce:    nonce,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
			Data:     hexutils.HexToBytes(strings.TrimPrefix(s.contractData.Bytecode, "0x")),
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
	ownerInitialBalance := (&big.Int{}).Mul(big.NewInt(1000_000), nibiCommon.TO_ATTO)
	s.assertERC20Balance(contractAddress, s.fundedAccEthAddr, ownerInitialBalance)

	// Querying contract: recipient balance should be 0
	recipientAddr := ethCommon.BytesToAddress(testutilcli.NewAccount(s.network, "contract_recipient"))
	s.assertERC20Balance(contractAddress, recipientAddr, big.NewInt(0))

	// Execute contract: send 1000 anibi to recipient
	sendAmount := (&big.Int{}).Mul(big.NewInt(1000), nibiCommon.TO_ATTO)
	input, err := s.contractData.Abi.Pack("transfer", recipientAddr, sendAmount)
	s.NoError(err)
	nonce, err = s.ethClient.NonceAt(context.Background(), s.fundedAccEthAddr, nil)
	s.NoError(err)
	tx, err = types.SignNewTx(
		s.fundedAccPrivateKey,
		signer,
		&types.LegacyTx{
			Nonce:    nonce,
			To:       &contractAddress,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
			Data:     input,
		})

	err = s.ethClient.SendTransaction(context.Background(), tx)
	s.NoError(err)
	s.NoError(s.network.WaitForNextBlock())

	// Querying contract: owner's balance should be 999_000 tokens
	ownerBalance := (&big.Int{}).Mul(big.NewInt(999_000), nibiCommon.TO_ATTO)
	s.assertERC20Balance(contractAddress, s.fundedAccEthAddr, ownerBalance)

	// Querying contract: recipient balance should be 1000 tokens
	recipientBalance := (&big.Int{}).Mul(big.NewInt(1000), nibiCommon.TO_ATTO)
	s.assertERC20Balance(contractAddress, recipientAddr, recipientBalance)
}

func (s *IntegrationSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func getContractData() *ContractJson {
	pkgDir, _ := testutil.GetPackageDir()
	pathToModulePkg := path.Dir(pkgDir)
	pathToContractJson := pathToModulePkg + fmt.Sprintf("/rpcapi/fixture/%s", fixture.ERC20_CONTRACT_FILE)
	file, _ := os.ReadFile(pathToContractJson)
	var contractData ContractJson
	_ = json.Unmarshal(file, &contractData)
	return &contractData
}

func (s *IntegrationSuite) assertERC20Balance(
	contractAddress ethCommon.Address,
	userAddress ethCommon.Address,
	expectedBalance *big.Int,
) {
	input, err := s.contractData.Abi.Pack("balanceOf", userAddress)
	s.NoError(err)
	msg := ethereum.CallMsg{
		From: s.fundedAccEthAddr,
		To:   &contractAddress,
		Data: input,
	}
	recipientBalanceBeforeBytes, err := s.ethClient.CallContract(context.Background(), msg, nil)
	s.NoError(err)
	balance := new(big.Int).SetBytes(recipientBalanceBeforeBytes)
	s.Equal(expectedBalance.String(), balance.String())
}
