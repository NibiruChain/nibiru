package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"
	"github.com/NibiruChain/nibiru/v2/x/tokenfactory/cli"
	"github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdktestutilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

var (
	_ suite.SetupAllSuite    = (*TestSuite)(nil)
	_ suite.TearDownAllSuite = (*TestSuite)(nil)
)

type TestSuite struct {
	suite.Suite

	cfg     testnetwork.Config
	network *testnetwork.Network
	val     *testnetwork.Validator
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CmdSuiteLite))

	testutil.RetrySuiteRunIfDbClosed(t, func() {
		suite.Run(t, new(TestSuite))
	}, 2)
}

// TestTokenFactory: Runs the test suite with a deterministic order.
func (s *TestSuite) TestTokenFactory() {
	s.Run("CreateDenomTest", s.CreateDenomTest)
	s.Run("MintBurnTest", s.MintBurnTest)
	s.Run("ChangeAdminTest", s.ChangeAdminTest)
}

func (s *TestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())
	encodingConfig := app.MakeEncodingConfig()
	genState := genesis.NewTestGenesisState(encodingConfig.Codec)
	cfg := testnetwork.BuildNetworkConfig(genState)
	cfg.NumValidators = 1
	network, err := testnetwork.New(s.T(), s.T().TempDir(), cfg)
	s.NoError(err)

	s.cfg = cfg
	s.network = network
	s.val = network.Validators[0]
	s.NoError(s.network.WaitForNextBlock())
}

func (s *TestSuite) CreateDenomTest() {
	creator := s.val.Address
	createDenom := func(subdenom string, wantErr bool) {
		_, err := s.network.ExecTxCmd(
			cli.NewTxCmd(),
			creator, []string{"create-denom", subdenom})
		if wantErr {
			s.Require().Error(err)
			return
		}
		s.Require().NoError(err)
		s.NoError(s.network.WaitForNextBlock())
	}

	createDenom("nusd", false)
	createDenom("nusd", true) // Can't create the same denom twice.
	createDenom("stnibi", false)
	createDenom("stnusd", false)

	denomResp := new(types.QueryDenomsResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.CmdQueryDenoms(), []string{creator.String()}, denomResp,
		),
	)
	denoms := denomResp.Denoms
	wantDenoms := []string{
		types.TFDenom{Creator: creator.String(), Subdenom: "nusd"}.Denom().String(),
		types.TFDenom{Creator: creator.String(), Subdenom: "stnibi"}.Denom().String(),
		types.TFDenom{Creator: creator.String(), Subdenom: "stnusd"}.Denom().String(),
	}
	s.ElementsMatch(denoms, wantDenoms)
}

func (s *TestSuite) MintBurnTest() {
	creator := s.val.Address
	mint := func(coin string, mintTo string, wantErr bool) {
		mintToArg := fmt.Sprintf("--mint-to=%s", mintTo)
		_, err := s.network.ExecTxCmd(
			cli.NewTxCmd(), creator, []string{"mint", coin, mintToArg})
		if wantErr {
			s.Require().Error(err)
			return
		}
		s.Require().NoError(err)
		s.NoError(s.network.WaitForNextBlock())
	}

	burn := func(coin string, burnFrom string, wantErr bool) {
		burnFromArg := fmt.Sprintf("--burn-from=%s", burnFrom)
		_, err := s.network.ExecTxCmd(
			cli.NewTxCmd(), creator, []string{"burn", coin, burnFromArg})
		if wantErr {
			s.Require().Error(err)
			return
		}
		s.Require().NoError(err)
		s.NoError(s.network.WaitForNextBlock())
	}

	t := s.T()
	t.Log("mint successfully")
	denom := types.TFDenom{
		Creator:  creator.String(),
		Subdenom: "nusd",
	}
	coin := sdk.NewInt64Coin(denom.Denom().String(), 420)
	wantErr := false
	mint(coin.String(), creator.String(), wantErr) // happy

	t.Log("want error: unregistered denom")
	coin.Denom = "notadenom"
	wantErr = true
	mint(coin.String(), creator.String(), wantErr)
	burn(coin.String(), creator.String(), wantErr)

	t.Log("want error: invalid coin")
	mint("notacoin_231,,", creator.String(), wantErr)
	burn("notacoin_231,,", creator.String(), wantErr)

	t.Log(`want error: unable to parse "mint-to" or "burn-from"`)
	coin.Denom = denom.Denom().String()
	mint(coin.String(), "invalidAddr", wantErr)
	burn(coin.String(), "invalidAddr", wantErr)

	t.Log("burn successfully")
	coin.Denom = denom.Denom().String()
	wantErr = false
	burn(coin.String(), creator.String(), wantErr) // happy
}

func (s *TestSuite) ChangeAdminTest() {
	creator := s.val.Address
	admin := creator
	newAdmin := testutil.AccAddress()
	denom := types.TFDenom{
		Creator:  creator.String(),
		Subdenom: "stnibi",
	}

	s.T().Log("Verify current admin is creator")
	infoResp := new(types.QueryDenomInfoResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(), []string{"denom-info", denom.Denom().String()}, infoResp,
		),
	)
	s.Equal(infoResp.Admin, admin.String())

	s.T().Log("Change to a new admin")
	_, err := s.network.ExecTxCmd(
		cli.NewTxCmd(),
		admin, []string{"change-admin", denom.Denom().String(), newAdmin.String()})
	s.Require().NoError(err)

	s.T().Log("Verify new admin is in state")
	infoResp = new(types.QueryDenomInfoResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(), []string{"denom-info", denom.Denom().String()}, infoResp,
		),
	)
	s.Equal(infoResp.Admin, newAdmin.String())
}

func (s *TestSuite) TestQueryModuleParams() {
	paramResp := new(types.QueryParamsResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(), []string{"params"}, paramResp,
		),
	)
	s.Equal(paramResp.Params, types.DefaultModuleParams())
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

type CmdTestCase struct {
	name      string
	args      []string
	extraArgs []string
	wantErr   string
}

// Flags for broadcasting transactions
func (s *CmdSuiteLite) commonTxArgs() []string {
	return []string{
		"--yes=true", // skip confirmation
		"--broadcast-mode=sync",
		"--fees=1unibi",
		"--chain-id=test-chain",
	}
}

type CmdSuiteLite struct {
	suite.Suite

	keyring    keyring.Keyring
	testEncCfg testutilmod.TestEncodingConfig

	testAcc sdktestutil.TestAccount
}

func (s *CmdSuiteLite) SetupSuite() {
	s.testEncCfg = testutilmod.TestEncodingConfig(app.MakeEncodingConfig())
	s.keyring = keyring.NewInMemory(s.testEncCfg.Codec)

	testAccs := sdktestutil.CreateKeyringAccounts(s.T(), s.keyring, 1)
	s.testAcc = testAccs[0]
}

func (s *CmdSuiteLite) TestCmdSetDenomMetadata() {
	s.T().Log(`Create a valid metadata file as "metadata.json"`)
	tempDir := s.T().TempDir()
	metadataFile, err := os.CreateTemp(tempDir, "metadata.json")
	s.Require().NoError(err)
	defer metadataFile.Close()

	_, err = metadataFile.Write([]byte(`
{
  "description": "A short description of the token",
  "denom_units": [
    {
      "denom": "testdenom"
    },
    {
      "denom": "TEST",
      "exponent": 6
    }
  ],
  "base": "testdenom",
  "display": "TEST",
  "name": "Test Token",
  "symbol": "TEST"
}`),
	)
	s.Require().NoError(err)

	metadatFilePath := metadataFile.Name()

	testCases := []CmdTestCase{
		{
			name: "happy: set-denom-metadata",
			args: []string{
				"set-denom-metadata",
				metadatFilePath,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "",
		},
		{
			name: "happy: sudo-set-denom-metadata",
			args: []string{
				"sudo-set-denom-metadata",
				metadatFilePath,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "",
		},
		{
			name: "happy: template flag",
			args: []string{
				"set-denom-metadata",
				"args.json",
				"--template",
			},
			extraArgs: []string{},
			wantErr:   "",
		},
		{
			name: "happy: template flag sudo",
			args: []string{
				"sudo-set-denom-metadata",
				"args.json",
				"--template",
			},
			extraArgs: []string{},
			wantErr:   "",
		},
		{
			name: "sad: no FILE given",
			args: []string{
				"set-denom-metadata",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "accepts 1 arg(s), received 0",
		},
		{
			name: "sad: file does not exist",
			args: []string{
				"set-denom-metadata",
				"not-a-file.json",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "no such file or directory",
		},
	}

	for _, tc := range testCases {
		testOutput := new(bytes.Buffer)
		tc.RunTxCmd(
			s,
			cli.NewTxCmd(),
			testOutput,
		)
	}
}

func (tc CmdTestCase) NewCtx(s *CmdSuiteLite) sdkclient.Context {
	return sdkclient.Context{}.
		WithKeyring(s.keyring).
		WithTxConfig(s.testEncCfg.TxConfig).
		WithCodec(s.testEncCfg.Codec).
		WithClient(sdktestutilcli.MockTendermintRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(sdkclient.MockAccountRetriever{}).
		WithOutput(io.Discard).
		WithChainID("test-chain")
}

func (tc CmdTestCase) RunTxCmd(s *CmdSuiteLite, txCmd *cobra.Command, output io.Writer) {
	s.Run(tc.name, func() {
		ctx := svrcmd.CreateExecuteContext(context.Background())

		cmd := txCmd
		cmd.SetContext(ctx)
		cmd.SetOutput(output)
		args := append(tc.args, s.commonTxArgs()...)
		cmd.SetArgs(append(args, tc.extraArgs...))

		s.Require().NoError(sdkclient.SetCmdClientContextHandler(tc.NewCtx(s), cmd))

		err := cmd.Execute()
		if tc.wantErr != "" {
			s.Require().ErrorContains(err, tc.wantErr)
			return
		}
		s.Require().NoError(err)
	})
}

func (tc CmdTestCase) RunQueryCmd(s *CmdSuiteLite, queryCmd *cobra.Command, output io.Writer) {
	s.Run(tc.name, func() {
		ctx := svrcmd.CreateExecuteContext(context.Background())

		cmd := queryCmd
		cmd.SetContext(ctx)
		cmd.SetOutput(output)
		args := tc.args // don't append common tx args
		cmd.SetArgs(append(args, tc.extraArgs...))

		s.Require().NoError(sdkclient.SetCmdClientContextHandler(tc.NewCtx(s), cmd))

		err := cmd.Execute()
		if tc.wantErr != "" {
			s.Require().ErrorContains(err, tc.wantErr)
			return
		}
		s.Require().NoError(err)
	})
}
