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
	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testnetwork"
	"github.com/NibiruChain/nibiru/v2/x/tokenfactory/cli"
	"github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
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

	creator     sdk.AccAddress
	localnetCLI testnetwork.LocalnetCLI
}

const (
	localnetAdminChangeSubdenom = "adminchange"
)

func Test(t *testing.T) {
	suite.Run(t, new(CmdSuiteLite))
	suite.Run(t, new(TestSuite))
}

// TestTokenFactory: Runs the test suite with a deterministic order.
func (s *TestSuite) TestTokenFactory() {
	s.Run("CreateDenomTest", s.CreateDenomTest)
	s.Run("MintBurnTest", s.MintBurnTest)
	s.Run("ChangeAdminTest", s.ChangeAdminTest)
}

func (s *TestSuite) SetupSuite() {
	if err := nutil.EnsureLocalBlockchain(); err != nil {
		s.T().Skipf("skipping localnet-backed tokenfactory CLI tests: %v", err)
	}
	s.creator = nutil.LocalnetValAddr

	localnetCLI, err := testnetwork.NewLocalnetCLI()
	s.Require().NoError(err)
	s.localnetCLI = localnetCLI
}

func (s *TestSuite) CreateDenomTest() {
	wantDenoms := []string{
		s.ensureDenomExists("nusd"),
		s.ensureDenomExists("stnibi"),
		s.ensureDenomExists("stnusd"),
		s.ensureDenomExists(localnetAdminChangeSubdenom),
	}

	denoms := s.queryCreatorDenoms()
	for _, denom := range wantDenoms {
		s.Require().Contains(denoms, denom)
	}

	s.T().Log("duplicate create-denom should fail once the denom exists")
	s.Require().Error(s.execLocalTx("create-denom", "nusd"))
}

func (s *TestSuite) MintBurnTest() {
	creator := s.creator
	denom := s.ensureDenomExists("nusd")
	infoResp := s.queryDenomInfo(denom)
	s.Require().Equalf(infoResp.Admin, creator.String(),
		"skipping mint/burn: %s admin is %s, not %s",
		denom, infoResp.Admin, creator.String(),
	)

	mint := func(coin string, mintTo string, wantErr bool) {
		mintToArg := fmt.Sprintf("--mint-to=%s", mintTo)
		err := s.execLocalTx("mint", coin, mintToArg)
		if wantErr {
			s.Require().Error(err)
			return
		}
		s.Require().NoError(err)
	}

	burn := func(coin string, burnFrom string, wantErr bool) {
		burnFromArg := fmt.Sprintf("--burn-from=%s", burnFrom)
		err := s.execLocalTx("burn", coin, burnFromArg)
		if wantErr {
			s.Require().Error(err)
			return
		}
		s.Require().NoError(err)
	}

	t := s.T()
	t.Log("mint successfully")
	coin := sdk.NewInt64Coin(denom, 420)
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
	coin.Denom = denom
	mint(coin.String(), "invalidAddr", wantErr)
	burn(coin.String(), "invalidAddr", wantErr)

	t.Log("burn successfully")
	coin.Denom = denom
	wantErr = false
	burn(coin.String(), creator.String(), wantErr) // happy
}

func (s *TestSuite) ChangeAdminTest() {
	creator := s.creator
	denom := s.ensureDenomExists(localnetAdminChangeSubdenom)
	newAdmin := "nibi1cr6tg4cjvux00pj6zjqkh6d0jzg7mksaywxyl3"

	s.T().Log("query current admin")
	infoResp := s.queryDenomInfo(denom)
	switch infoResp.Admin {
	case creator.String():
		s.T().Log("Change to a fixed localnet admin")
		s.Require().NoError(s.execLocalTx(
			"change-admin",
			denom,
			newAdmin,
		))
	case newAdmin:
		s.T().Log("admin already changed on a previous localnet test run")
	default:
		s.T().Fatalf(
			"skipping change-admin: %s admin is %s, expected %s or %s",
			denom, infoResp.Admin, creator.String(), newAdmin,
		)
	}

	s.T().Log("Verify new admin is in state")
	infoResp = s.queryDenomInfo(denom)
	s.Equal(newAdmin, infoResp.Admin)
}

func (s *TestSuite) TestQueryModuleParams() {
	resp := new(types.QueryParamsResponse)
	s.Require().NoError(s.execLocalQuery(resp, "params"))
	s.Positive(resp.Params.DenomCreationGasConsume)
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("leaving localnet state in place")
}

func (s *TestSuite) tokenfactoryDenom(subdenom string) string {
	return types.TFDenom{
		Creator:  s.creator.String(),
		Subdenom: subdenom,
	}.Denom().String()
}

func (s *TestSuite) ensureDenomExists(subdenom string) string {
	denom := s.tokenfactoryDenom(subdenom)
	if s.hasDenom(denom) {
		return denom
	}

	err := s.execLocalTx("create-denom", subdenom)
	if err != nil && !s.hasDenom(denom) {
		s.Require().NoError(err)
	}
	s.Require().Contains(s.queryCreatorDenoms(), denom)
	return denom
}

func (s *TestSuite) hasDenom(denom string) bool {
	denoms := s.queryCreatorDenoms()
	for _, got := range denoms {
		if got == denom {
			return true
		}
	}
	return false
}

func (s *TestSuite) queryCreatorDenoms() []string {
	resp := new(types.QueryDenomsResponse)
	s.Require().NoError(s.execLocalQuery(resp, "denoms", s.creator.String()))
	return resp.Denoms
}

func (s *TestSuite) queryDenomInfo(denom string) *types.QueryDenomInfoResponse {
	resp := new(types.QueryDenomInfoResponse)
	s.Require().NoError(s.execLocalQuery(resp, "denom-info", denom))
	return resp
}

func (s *TestSuite) execLocalQuery(
	result codec.ProtoMarshaler,
	args ...string,
) error {
	cmd := cli.NewQueryCmd()
	s.T().Log(s.localnetCLI.RenderQueryCmd(cmd, args))
	return s.localnetCLI.ExecQueryCmd(cmd, args, result)
}

func (s *TestSuite) execLocalTx(args ...string) error {
	cmd := cli.NewTxCmd()
	s.T().Log(s.localnetCLI.RenderTxCmd(cmd, args))
	_, err := s.localnetCLI.ExecTxCmd(cmd, args)
	return err
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
