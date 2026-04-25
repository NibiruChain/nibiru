package gosdk_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/v2/gosdk"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/denoms"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testnetwork"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// --------------------------------------------------
// NibiruClientSuite
// --------------------------------------------------

var (
	_ suite.SetupAllSuite    = (*Suite)(nil)
	_ suite.TearDownAllSuite = (*Suite)(nil)
)

type Suite struct {
	suite.Suite

	nibiruSdk    *gosdk.NibiruSDK
	grpcConn     *grpc.ClientConn
	localnetCLI  testnetwork.LocalnetCLI
	localnetInfo gosdk.NetworkInfo
	from         sdk.AccAddress
}

func TestGosdk(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) RPCEndpoint() string {
	return s.localnetInfo.TmRpcEndpoint
}

// SetupSuite implements the suite.SetupAllSuite interface. This function runs
// prior to all of the other tests in the suite.
func (s *Suite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())

	s.Run("DoTestGetGrpcConnection_NoNetwork", s.DoTestGetGrpcConnection_NoNetwork)

	if err := nutil.EnsureLocalBlockchain(); err != nil {
		s.T().Skipf("skipping localnet-backed Go SDK tests: %v", err)
	}

	s.localnetInfo = gosdk.NETWORK_INFO_DEFAULT
	s.from = nutil.LocalnetValAddr

	localnetCLI, err := testnetwork.NewLocalnetCLI()
	s.Require().NoError(err)
	s.localnetCLI = localnetCLI

	grpcConn, err := gosdk.GetGRPCConnection(
		s.localnetInfo.GrpcEndpoint,
		true,
		5,
	)
	s.Require().NoError(err)
	s.Require().NotNil(grpcConn)
	s.grpcConn = grpcConn
}

func (s *Suite) ConnectGrpc() {
	grpcConn, err := gosdk.GetGRPCConnection(s.localnetInfo.GrpcEndpoint, true, 5)
	s.NoError(err)
	s.NotNil(grpcConn)
	s.grpcConn = grpcConn
}

func (s *Suite) TestNewNibiruSdk() {
	nibiruSdk, err := gosdk.NewNibiruSdk(
		s.localnetInfo.CmtChainID,
		s.grpcConn,
		s.RPCEndpoint(),
	)
	s.NoError(err)
	s.nibiruSdk = &nibiruSdk
	s.nibiruSdk.Keyring = s.localnetCLI.ClientCtx.Keyring

	s.Run("DoTestBroadcastMsgs", func() {
		txHashHex := s.DoTestBroadcastMsgs()
		s.Require().NoError(s.localnetCLI.WaitForNextBlock())
		_, err := s.localnetCLI.WaitForTx(txHashHex)
		s.Require().NoError(err)
	})
	s.Run("DoTestBroadcastMsgsGrpc", func() {
		txHashHex := s.DoTestBroadcastMsgsGrpc()
		s.Require().NoError(s.localnetCLI.WaitForNextBlock())
		_, err := s.localnetCLI.WaitForTx(txHashHex)
		s.Require().NoError(err)
	})
	s.Run("DoTestNewQueryClient", func() {
		s.NotNil(s.nibiruSdk.Querier)
		s.NotNil(s.nibiruSdk.Querier.ClientConn)
	})
}

// FIXME: Q: What is the node home for a local validator?
func (s *Suite) UsefulPrints() {
	fmt.Printf("localnet grpc endpoint: %v\n", s.localnetInfo.GrpcEndpoint)
	fmt.Printf("localnet rpc endpoint: %v\n", s.localnetInfo.TmRpcEndpoint)
	fmt.Printf("localnet keyring dir: %v\n", s.localnetCLI.ClientCtx.KeyringDir)
}

func (s *Suite) AssertTxResponseSuccess(txResp *sdk.TxResponse) (txHashHex string) {
	s.NotNil(txResp)
	s.EqualValues(txResp.Code, 0)
	return txResp.TxHash
}

func (s *Suite) msgSendVars() (from, to sdk.AccAddress, amt sdk.Coins, msgSend sdk.Msg) {
	from = s.from
	to = testutil.AccAddress()
	amt = sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 420))
	msgSend = banktypes.NewMsgSend(from, to, amt)
	return from, to, amt, msgSend
}

func (s *Suite) DoTestBroadcastMsgs() (txHashHex string) {
	from, _, _, msgSend := s.msgSendVars()
	txResp, err := s.nibiruSdk.BroadcastMsgs(
		from, msgSend,
	)
	s.NoError(err)
	return s.AssertTxResponseSuccess(txResp)
}

func (s *Suite) DoTestBroadcastMsgsGrpc() (txHashHex string) {
	from, _, _, msgSend := s.msgSendVars()
	txResp, err := s.nibiruSdk.BroadcastMsgsGrpc(
		from, msgSend,
	)
	s.NoError(err)
	txHashHex = s.AssertTxResponseSuccess(txResp)

	base := 10
	txRespCode := strconv.FormatUint(uint64(txResp.Code), base)
	s.EqualValuesf(txResp.Code, 0,
		"code: %v\nraw log: %s", txRespCode, txResp.RawLog)
	return txHashHex
}

func (s *Suite) TearDownSuite() {
	if s.grpcConn != nil {
		s.Require().NoError(s.grpcConn.Close())
	}
	s.T().Log("leaving localnet state in place")
}

func (s *Suite) DoTestGetGrpcConnection_NoNetwork() {
	grpcConn, err := gosdk.GetGRPCConnection(
		gosdk.NETWORK_INFO_DEFAULT.GrpcEndpoint+"notendpoint", true, 2,
	)
	s.Error(err)
	s.Nil(grpcConn)
}
