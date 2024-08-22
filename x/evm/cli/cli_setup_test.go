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

	"github.com/NibiruChain/nibiru/v2/x/evm/cli"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmmodule"
)

type Suite struct {
	suite.Suite

	keyring   keyring.Keyring
	encCfg    testutilmod.TestEncodingConfig
	baseCtx   sdkclient.Context
	clientCtx sdkclient.Context

	testAcc sdktestutil.TestAccount
}

func (s *Suite) SetupSuite() {
	s.encCfg = testutilmod.MakeTestEncodingConfig(evmmodule.AppModuleBasic{})
	s.keyring = keyring.NewInMemory(s.encCfg.Codec)
	s.baseCtx = sdkclient.Context{}.
		WithKeyring(s.keyring).
		WithTxConfig(s.encCfg.TxConfig).
		WithCodec(s.encCfg.Codec).
		WithClient(sdktestutilcli.MockTendermintRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(sdkclient.MockAccountRetriever{}).
		WithOutput(io.Discard).
		WithChainID("test-chain")

	s.clientCtx = s.baseCtx

	testAccs := sdktestutil.CreateKeyringAccounts(s.T(), s.keyring, 1)
	s.testAcc = testAccs[0]
}

func TestSuite(t *testing.T) {
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

func (tc TestCase) NewCtx(s *Suite) sdkclient.Context {
	return s.baseCtx
}

func (tc TestCase) RunTxCmd(s *Suite) {
	s.Run(tc.name, func() {
		ctx := svrcmd.CreateExecuteContext(context.Background())

		cmd := cli.GetTxCmd()
		cmd.SetContext(ctx)
		args := append(tc.args, commonTxArgs()...)
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

func (tc TestCase) RunQueryCmd(s *Suite) {
	s.Run(tc.name, func() {
		ctx := svrcmd.CreateExecuteContext(context.Background())

		cmd := cli.GetQueryCmd()
		cmd.SetContext(ctx)
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
