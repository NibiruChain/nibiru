package gosdk_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/v2/gosdk"
	"github.com/NibiruChain/nibiru/v2/gosdk/gosdktest"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"

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

	nibiruSdk *gosdk.NibiruSDK
	grpcConn  *grpc.ClientConn
	cfg       *testnetwork.Config
	network   *testnetwork.Network
	val       *testnetwork.Validator
}

func TestGosdk(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) RPCEndpoint() string {
	return s.val.RPCAddress
}

// SetupSuite implements the suite.SetupAllSuite interface. This function runs
// prior to all of the other tests in the suite.
func (s *Suite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())

	s.Run("DoTestGetGrpcConnection_NoNetwork", s.DoTestGetGrpcConnection_NoNetwork)

	nibiru, err := gosdktest.CreateBlockchain(&s.Suite)
	s.NoError(err)
	s.network = nibiru.Network
	s.cfg = nibiru.Cfg
	s.val = nibiru.Val
	s.grpcConn = nibiru.GrpcConn

	s.NotNil(
		s.val.Querier,
		"NewQuerier should be used in network setup",
	)
}

func (s *Suite) ConnectGrpc() {
	grpcConn, err := testnetwork.ConnectGrpcToVal(s.val)
	s.NoError(err)
	s.NotNil(grpcConn)
	s.grpcConn = grpcConn
}

func (s *Suite) TestNewNibiruSdk() {
	rpcEndpt := s.val.RPCAddress
	nibiruSdk, err := gosdk.NewNibiruSdk(s.cfg.ChainID, s.grpcConn, rpcEndpt)
	s.NoError(err)
	s.nibiruSdk = &nibiruSdk
	s.nibiruSdk.Keyring = s.val.ClientCtx.Keyring

	s.Run("DoTestBroadcastMsgs", func() {
		s.DoTestBroadcastMsgs()
	})
	s.Run("DoTestBroadcastMsgsGrpc", func() {
		for t := 0; t < 4; t++ {
			s.network.WaitForNextBlock()
		}
		s.DoTestBroadcastMsgsGrpc()
	})
	s.Run("DoTestNewQueryClient", func() {
		s.NotNil(s.val.Querier)
	})
}

// FIXME: Q: What is the node home for a local validator?
func (s *Suite) UsefulPrints() {
	tmCfgRootDir := s.val.Ctx.Config.RootDir
	fmt.Printf("tmCfgRootDir: %v\n", tmCfgRootDir)
	fmt.Printf("s.val.Dir: %v\n", s.val.Dir)
	fmt.Printf("s.val.ClientCtx.KeyringDir: %v\n", s.val.ClientCtx.KeyringDir)
}

func (s *Suite) AssertTxResponseSuccess(txResp *sdk.TxResponse) (txHashHex string) {
	s.NotNil(txResp)
	s.EqualValues(txResp.Code, 0)
	return txResp.TxHash
}

func (s *Suite) msgSendVars() (from, to sdk.AccAddress, amt sdk.Coins, msgSend sdk.Msg) {
	from = s.val.Address
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
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *Suite) DoTestGetGrpcConnection_NoNetwork() {
	grpcConn, err := gosdk.GetGRPCConnection(
		gosdk.NETWORK_INFO_DEFAULT.GrpcEndpoint+"notendpoint", true, 2,
	)
	s.Error(err)
	s.Nil(grpcConn)
}
