package gosdk_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/gosdk"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/cli"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// --------------------------------------------------
// NibiruClientSuite
// --------------------------------------------------

var (
	_ suite.SetupAllSuite    = (*TestSuite)(nil)
	_ suite.TearDownAllSuite = (*TestSuite)(nil)
)

type TestSuite struct {
	suite.Suite

	nibiruSdk *gosdk.NibiruSDK
	grpcConn  *grpc.ClientConn
	cfg       *cli.Config
	network   *cli.Network
	val       *cli.Validator
}

func TestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) RPCEndpoint() string {
	return s.val.RPCAddress
}

// SetupSuite implements the suite.SetupAllSuite interface. This function runs
// prior to all of the other tests in the suite.
func (s *TestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())

	s.Run("DoTestGetGrpcConnection_NoNetwork", s.DoTestGetGrpcConnection_NoNetwork)

	nibiru, err := gosdk.CreateBlockchain(s.T())
	s.NoError(err)
	s.network = nibiru.Network
	s.cfg = nibiru.Cfg
	s.val = nibiru.Val
	s.grpcConn = nibiru.GrpcConn
}

func ConnectGrpcToVal(val *cli.Validator) (*grpc.ClientConn, error) {
	grpcUrl := val.AppConfig.GRPC.Address
	return gosdk.GetGRPCConnection(
		grpcUrl, true, 5,
	)
}

func (s *TestSuite) ConnectGrpc() {
	grpcConn, err := ConnectGrpcToVal(s.val)
	s.NoError(err)
	s.NotNil(grpcConn)
	s.grpcConn = grpcConn
}

func (s *TestSuite) DoTestNewQueryClient() {
	_, err := gosdk.NewQuerier(s.grpcConn)
	s.NoError(err)
}

func (s *TestSuite) TestNewNibiruSdk() {
	rpcEndpt := s.val.RPCAddress
	nibiruSdk, err := gosdk.NewNibiruSdk(s.cfg.ChainID, s.grpcConn, rpcEndpt)
	s.NoError(err)
	s.nibiruSdk = &nibiruSdk

	s.nibiruSdk.Keyring = s.val.ClientCtx.Keyring

	s.Run("DoTestBroadcastMsgs", func() { s.DoTestBroadcastMsgs() })
	s.Run("DoTestBroadcastMsgsGrpc", func() {
		s.NoError(s.network.WaitForNextBlock())
		s.DoTestBroadcastMsgsGrpc()
	})
	s.Run("DoTestNewQueryClient", s.DoTestNewQueryClient)
	s.Run("DoTestSequenceExpectations", s.DoTestSequenceExpectations)
}

// FIXME: Q: What is the node home for a local validator?
func (s *TestSuite) UsefulPrints() {
	tmCfgRootDir := s.val.Ctx.Config.RootDir
	fmt.Printf("tmCfgRootDir: %v\n", tmCfgRootDir)
	fmt.Printf("s.val.Dir: %v\n", s.val.Dir)
	fmt.Printf("s.val.ClientCtx.KeyringDir: %v\n", s.val.ClientCtx.KeyringDir)
}

func (s *TestSuite) AssertTxResponseSuccess(txResp *sdk.TxResponse) (txHashHex string) {
	s.NotNil(txResp)
	s.EqualValues(txResp.Code, 0)
	return txResp.TxHash
}

func (s *TestSuite) msgSendVars() (from, to sdk.AccAddress, amt sdk.Coins, msgSend sdk.Msg) {
	from = s.val.Address
	to = testutil.AccAddress()
	amt = sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 420))
	msgSend = banktypes.NewMsgSend(from, to, amt)
	return from, to, amt, msgSend
}

func (s *TestSuite) DoTestBroadcastMsgs() (txHashHex string) {
	from, _, _, msgSend := s.msgSendVars()
	txResp, err := s.nibiruSdk.BroadcastMsgs(
		from, msgSend,
	)
	s.NoError(err)
	return s.AssertTxResponseSuccess(txResp)
}

func (s *TestSuite) DoTestBroadcastMsgsGrpc() (txHashHex string) {
	from, _, _, msgSend := s.msgSendVars()
	txResp, err := s.nibiruSdk.BroadcastMsgsGrpc(
		from, msgSend,
	)
	s.NoError(err)
	txHashHex = s.AssertTxResponseSuccess(txResp)

	base := 10
	var txRespCode string = strconv.FormatUint(uint64(txResp.Code), base)
	s.EqualValuesf(txResp.Code, 0,
		"code: %v\nraw log: %s", txRespCode, txResp.RawLog)
	return txHashHex
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *TestSuite) DoTestGetGrpcConnection_NoNetwork() {
	grpcConn, err := gosdk.GetGRPCConnection(
		gosdk.DefaultNetworkInfo.GrpcEndpoint+"notendpoint", true, 2,
	)
	s.Error(err)
	s.Nil(grpcConn)

	_, err = gosdk.NewQuerier(grpcConn)
	s.Error(err)
}
