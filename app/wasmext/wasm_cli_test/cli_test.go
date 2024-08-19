package wasm_cli_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	wasmcli "github.com/CosmWasm/wasmd/x/wasm/client/cli"

	"cosmossdk.io/math"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"
)

// commonArgs is args for CLI test commands.
var commonArgs = []string{
	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
	fmt.Sprintf("--%s=%s", flags.FlagFees,
		sdk.NewCoins(sdk.NewCoin(denoms.NIBI, math.NewInt(10_000_000))).String()),
}

var _ suite.TearDownAllSuite = (*TestSuite)(nil)

type TestSuite struct {
	suite.Suite

	cfg     testnetwork.Config
	network *testnetwork.Network
}

func (s *TestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())
	testapp.EnsureNibiruPrefix()

	encodingConfig := app.MakeEncodingConfig()
	genesisState := genesis.NewTestGenesisState(encodingConfig)
	s.cfg = testnetwork.BuildNetworkConfig(genesisState)
	network, err := testnetwork.New(s.T(), s.T().TempDir(), s.cfg)
	s.Require().NoError(err)

	s.network = network
	s.Require().NoError(s.network.WaitForNextBlock())
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *TestSuite) TestWasmHappyPath() {
	s.requiredDeployedContractsLen(0)

	_, err := s.deployWasmContract("testdata/cw_nameservice.wasm")
	s.Require().NoError(err)

	err = s.network.WaitForNextBlock()
	s.Require().NoError(err)

	s.requiredDeployedContractsLen(1)
}

// deployWasmContract deploys a wasm contract located in path.
func (s *TestSuite) deployWasmContract(path string) (uint64, error) {
	val := s.network.Validators[0]
	codec := val.ClientCtx.Codec

	args := []string{
		path,
		"--from", val.Address.String(),
		"--gas", "11000000",
	}
	args = append(args, commonArgs...)

	cmd := wasmcli.StoreCodeCmd()
	out, err := cli.ExecTestCLICmd(val.ClientCtx, cmd, args)
	if err != nil {
		return 0, err
	}
	s.Require().NoError(s.network.WaitForNextBlock())

	resp := &sdk.TxResponse{}
	err = codec.UnmarshalJSON(out.Bytes(), resp)
	if err != nil {
		return 0, err
	}

	resp, err = testnetwork.QueryTx(val.ClientCtx, resp.TxHash)
	if err != nil {
		return 0, err
	}

	decodedResult, err := hex.DecodeString(resp.Data)
	if err != nil {
		return 0, err
	}

	respData := sdk.TxMsgData{}
	err = codec.Unmarshal(decodedResult, &respData)
	if err != nil {
		return 0, err
	}

	if len(respData.MsgResponses) < 1 {
		return 0, fmt.Errorf("no data found in response")
	}

	var storeCodeResponse types.MsgStoreCodeResponse
	err = codec.Unmarshal(respData.MsgResponses[0].Value, &storeCodeResponse)
	if err != nil {
		return 0, err
	}

	return storeCodeResponse.CodeID, nil
}

// requiredDeployedContractsLen checks the number of deployed contracts.
func (s *TestSuite) requiredDeployedContractsLen(total int) {
	val := s.network.Validators[0]
	var queryCodeResponse types.QueryCodesResponse
	err := testnetwork.ExecQuery(
		val.ClientCtx,
		wasmcli.GetCmdListCode(),
		[]string{},
		&queryCodeResponse,
	)
	s.Require().NoError(err)
	s.Require().Len(queryCodeResponse.CodeInfos, total)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
