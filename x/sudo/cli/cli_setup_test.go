package cli_test

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdktestutilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/NibiruChain/nibiru/v2/x/sudo/cli"
	"github.com/NibiruChain/nibiru/v2/x/sudo/sudomodule"
)

type TestVars struct {
	Keyring keyring.Keyring
	EncCfg  testutilmod.TestEncodingConfig
	TestAcc sdktestutil.TestAccount
}

func SetupTestVars(t *testing.T) TestVars {
	encCfg := testutilmod.MakeTestEncodingConfig(sudomodule.AppModuleBasic{})
	keyring := keyring.NewInMemory(encCfg.Codec)
	testAccs := sdktestutil.CreateKeyringAccounts(t, keyring, 1)

	return TestVars{
		Keyring: keyring,
		EncCfg:  encCfg,
		TestAcc: testAccs[0],
	}
}

type Suite struct {
	suite.Suite
}

func TestSudoCLI(t *testing.T) {
	suite.Run(t, new(Suite))
}

// Flags for broadcasting transactions
func commonTxArgs() []string {
	return []string{
		"--yes=true", // skip confirmation
		"--broadcast-mode=sync",
		"--fees=1unibi",
		"--chain-id=test-chain",
	}
}

type TestCase struct {
	name      string
	args      []string
	extraArgs []string
	wantErr   string
}

func (tc TestCase) NewCtx(testVars TestVars) sdkclient.Context {
	return sdkclient.Context{}.
		WithKeyring(testVars.Keyring).
		WithTxConfig(testVars.EncCfg.TxConfig).
		WithCodec(testVars.EncCfg.Codec).
		WithClient(sdktestutilcli.MockTendermintRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(sdkclient.MockAccountRetriever{}).
		WithOutput(io.Discard).
		WithChainID("test-chain")
}

func (tc TestCase) RunTxCmd(s *Suite, testVars TestVars) {
	s.Run(tc.name, func() {
		ctx := svrcmd.CreateExecuteContext(context.Background())

		cmd := cli.GetTxCmd()
		cmd.SetContext(ctx)
		args := append(tc.args, commonTxArgs()...)
		cmd.SetArgs(append(args, tc.extraArgs...))

		s.Require().NoError(sdkclient.SetCmdClientContextHandler(tc.NewCtx(testVars), cmd))

		err := cmd.Execute()
		if tc.wantErr != "" {
			s.Require().ErrorContains(err, tc.wantErr)
			return
		}
		s.Require().NoError(err)
	})
}

func (tc TestCase) RunQueryCmd(s *Suite, testVars TestVars) {
	s.Run(tc.name, func() {
		ctx := svrcmd.CreateExecuteContext(context.Background())

		cmd := cli.GetQueryCmd()
		cmd.SetContext(ctx)
		args := tc.args // don't append common tx args
		cmd.SetArgs(append(args, tc.extraArgs...))

		s.Require().NoError(sdkclient.SetCmdClientContextHandler(tc.NewCtx(testVars), cmd))

		err := cmd.Execute()
		if tc.wantErr != "" {
			s.Require().ErrorContains(err, tc.wantErr)
			return
		}
		s.Require().NoError(err)
	})
}
