package rpcapi_test

import (
	"context"
	"fmt"
	nibiCommon "github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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

	ethClient *ethclient.Client
}

func TestSuite_IntegrationSuite_RunAll(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

// ———————————————————————————————————————————————————————————————————
// IntegrationSuite - Setup
// ———————————————————————————————————————————————————————————————————

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

// Test_Balance EVM method: eth_getBalance
func (s *IntegrationSuite) Test_Balance() {
	val := s.network.Validators[0]

	expectedBalance := 123 * nibiCommon.TO_MICRO
	funds := sdk.NewCoins(
		sdk.NewInt64Coin(denoms.NIBI, expectedBalance),
	)
	testAcc := testutilcli.NewAccount(s.network, "ethuser")
	s.NoError(testutilcli.FillWalletFromValidator(
		testAcc, funds, val, denoms.NIBI,
	))
	ethAcc := ethCommon.BytesToAddress(testAcc.Bytes())

	balance, err := s.ethClient.BalanceAt(context.Background(), ethAcc, nil)
	s.NoError(err)

	// TODO: this fails, probably querying wrong eth account
	s.Equal(expectedBalance, *balance)
}

func (s *IntegrationSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}
