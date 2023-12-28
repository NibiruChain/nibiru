package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	"github.com/stretchr/testify/suite"

	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdktestutilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	devgas "github.com/NibiruChain/nibiru/x/devgas/v1"
	"github.com/NibiruChain/nibiru/x/devgas/v1/client/cli"
)

// CLITestSuite: Tests all tx commands for the module.
type CLITestSuite struct {
	suite.Suite

	keyring   keyring.Keyring
	encCfg    testutilmod.TestEncodingConfig
	baseCtx   sdkclient.Context
	clientCtx sdkclient.Context

	testAcc sdktestutil.TestAccount
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

// Runs once before the entire test suite.
func (s *CLITestSuite) SetupSuite() {
	s.encCfg = testutilmod.MakeTestEncodingConfig(devgas.AppModuleBasic{})
	s.keyring = keyring.NewInMemory(s.encCfg.Codec)
	s.baseCtx = sdkclient.Context{}.
		WithKeyring(s.keyring).
		WithTxConfig(s.encCfg.TxConfig).
		WithCodec(s.encCfg.Codec).
		WithClient(sdktestutilcli.MockTendermintRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(sdkclient.MockAccountRetriever{}).
		WithOutput(io.Discard).
		WithChainID("test-chain")

	var outBuf bytes.Buffer
	ctxGen := func() sdkclient.Context {
		bz, _ := s.encCfg.Codec.Marshal(&sdk.TxResponse{})
		c := sdktestutilcli.NewMockTendermintRPC(abci.ResponseQuery{
			Value: bz,
		})
		return s.baseCtx.WithClient(c)
	}
	s.clientCtx = ctxGen().WithOutput(&outBuf)

	testAccs := sdktestutil.CreateKeyringAccounts(s.T(), s.keyring, 1)
	s.testAcc = testAccs[0]
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

func (tc TestCase) NewCtx(s *CLITestSuite) sdkclient.Context {
	return s.baseCtx
}

func (tc TestCase) Run(s *CLITestSuite) {
	s.Run(tc.name, func() {
		ctx := svrcmd.CreateExecuteContext(context.Background())

		cmd := cli.NewTxCmd()
		cmd.SetContext(ctx)
		args := append(tc.args, commonTxArgs()...)
		cmd.SetArgs(append(args, tc.extraArgs...))

		s.Require().NoError(sdkclient.SetCmdClientContextHandler(tc.NewCtx(s), cmd))

		err := cmd.Execute()
		if tc.wantErr != "" {
			s.Require().Error(err)
			s.ErrorContains(err, tc.wantErr)
			return
		}
		s.Require().NoError(err)
	})
}

func (s *CLITestSuite) TestCmdRegisterFeeShare() {
	_, addrs := testutil.PrivKeyAddressPairs(3)

	testCases := []TestCase{
		{
			name:      "happy path: devgas register",
			args:      []string{"register", addrs[0].String(), addrs[1].String()},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "",
		},
		{
			name: "sad: fee payer",
			args: []string{"register", addrs[0].String(), addrs[1].String()},
			extraArgs: []string{
				fmt.Sprintf("--from=%s", s.testAcc.Address),
				fmt.Sprintf("--fee-payer=%s", "invalid-fee-payer"),
			},
			wantErr: "decoding bech32 failed",
		},
		{
			name: "sad: contract addr",
			args: []string{"register", "sadcontract", addrs[1].String()},
			extraArgs: []string{
				fmt.Sprintf("--from=%s", s.testAcc.Address),
			},
			wantErr: "invalid contract address",
		},
		{
			name: "sad: withdraw addr",
			args: []string{"register", addrs[0].String(), "sadwithdraw"},
			extraArgs: []string{
				fmt.Sprintf("--from=%s", s.testAcc.Address),
			},
			wantErr: "invalid withdraw address",
		},
	}

	for _, tc := range testCases {
		tc.Run(s)
	}
}

func (s *CLITestSuite) TestCmdCancelFeeShare() {
	_, addrs := testutil.PrivKeyAddressPairs(1)
	testCases := []TestCase{
		{
			name:      "happy path: devgas cancel",
			args:      []string{"cancel", addrs[0].String()},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "",
		},
		{
			name: "sad: fee payer",
			args: []string{"cancel", addrs[0].String()},
			extraArgs: []string{
				fmt.Sprintf("--from=%s", s.testAcc.Address),
				fmt.Sprintf("--fee-payer=%s", "invalid-fee-payer"),
			},
			wantErr: "decoding bech32 failed",
		},
		{
			name: "sad: contract addr",
			args: []string{"cancel", "sadcontract"},
			extraArgs: []string{
				fmt.Sprintf("--from=%s", s.testAcc.Address),
			},
			wantErr: "invalid deployer address",
		},
	}

	for _, tc := range testCases {
		tc.Run(s)
	}
}

func (s *CLITestSuite) TestCmdUpdateFeeShare() {
	_, addrs := testutil.PrivKeyAddressPairs(3)

	testCases := []TestCase{
		{
			name:      "happy path: devgas update",
			args:      []string{"update", addrs[0].String(), addrs[1].String()},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "",
		},
		{
			name: "sad: fee payer",
			args: []string{"update", addrs[0].String(), addrs[1].String()},
			extraArgs: []string{
				fmt.Sprintf("--from=%s", s.testAcc.Address),
				fmt.Sprintf("--fee-payer=%s", "invalid-fee-payer"),
			},
			wantErr: "decoding bech32 failed",
		},
		{
			name: "sad: contract addr",
			args: []string{"update", "sadcontract", addrs[1].String()},
			extraArgs: []string{
				fmt.Sprintf("--from=%s", s.testAcc.Address),
			},
			wantErr: "invalid contract",
		},
		{
			name: "sad: new withdraw addr",
			args: []string{"update", addrs[0].String(), "saddeployer"},
			extraArgs: []string{
				fmt.Sprintf("--from=%s", s.testAcc.Address),
			},
			wantErr: "invalid withdraw address",
		},
	}

	for _, tc := range testCases {
		tc.Run(s)
	}
}
