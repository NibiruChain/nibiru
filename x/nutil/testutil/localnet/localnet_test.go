package localnet_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	nibidcmd "github.com/NibiruChain/nibiru/v2/cmd/nibid/impl"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/localnet"
)

func Test(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	suite.Suite

	localnetCLI localnet.CLI
}

var (
	_ suite.SetupAllSuite    = (*Suite)(nil)
	_ suite.TearDownAllSuite = (*Suite)(nil)
)

func (s *Suite) SetupSuite() {
	if err := nutil.EnsureLocalBlockchain(); err != nil {
		s.T().Skipf("skipping localnet tests since the chain isn't active: %v", err)
	}

	localnetCLI, err := localnet.NewCLI()
	s.Require().NoError(err)
	s.localnetCLI = localnetCLI
}

func (s *Suite) TearDownSuite() {
	s.Require().NoError(s.localnetCLI.Close())
}

func (s *Suite) TestLocalnetCLIRenderQueryCmd() {
	localnetCLI := localnet.CLI{
		FromName: localnet.KeyName,
		FromAddr: nutil.LocalnetValAddr,
		NodeURI:  localnet.NodeURI,
		TxFee:    localnet.TxFeeDefault,
		TxGas:    localnet.TxGasDefault,
	}

	cmd := &cobra.Command{Use: "tokenfactory"}
	got := localnetCLI.RenderQueryCmd(cmd, []string{"denom-info", "factory/nibi1abc/sub denom"})

	s.Require().Equal(
		"nibid q tokenfactory denom-info 'factory/nibi1abc/sub denom' --output=json --node=http://localhost:26657",
		got,
	)
}

func (s *Suite) TestLocalnetCLIRenderTxCmd() {
	localnetCLI := localnet.CLI{
		FromName: localnet.KeyName,
		FromAddr: nutil.LocalnetValAddr,
		NodeURI:  localnet.NodeURI,
		TxFee:    localnet.TxFeeDefault,
		TxGas:    localnet.TxGasDefault,
	}

	cmd := &cobra.Command{Use: "tokenfactory"}
	got := localnetCLI.RenderTxCmd(cmd, []string{"create-denom", "sub denom"})

	s.Require().Equal(
		"nibid tx tokenfactory create-denom 'sub denom' --from=validator --fees=1000unibi --gas=5000000 --yes=true --broadcast-mode=sync --chain-id=nibiru-localnet-0 --keyring-backend=test --node=http://localhost:26657 --output=json",
		got,
	)
}

func (s *Suite) TestLocalnetCLIRenderTxCmdWithOptions() {
	localnetCLI := localnet.CLI{
		FromName: localnet.KeyName,
		FromAddr: nutil.LocalnetValAddr,
		NodeURI:  localnet.NodeURI,
		TxFee:    localnet.TxFeeDefault,
		TxGas:    localnet.TxGasDefault,
	}

	cmd := &cobra.Command{Use: "wasm"}
	got := localnetCLI.RenderTxCmd(
		cmd,
		[]string{"store", "contract.wasm"},
		localnet.WithTxGas("auto"),
		localnet.WithTxGasAdjustment("1.5"),
		localnet.WithTxFees("10000000unibi"),
	)

	s.Require().Equal(
		"nibid tx wasm store contract.wasm --from=validator --fees=10000000unibi --gas=auto --yes=true --broadcast-mode=sync --chain-id=nibiru-localnet-0 --keyring-backend=test --node=http://localhost:26657 --output=json --gas-adjustment=1.5",
		got,
	)
}

func (s *Suite) TestNewCLIExposesTypedEvmRpcAPIs() {
	cli := s.localnetCLI
	s.Require().NotNil(cli.EvmRpc.Eth)
	s.Require().NotNil(cli.EvmRpc.Filters)
	s.Require().NotNil(cli.EvmRpc.Net)
	s.Require().NotNil(cli.EvmRpc.Debug)
}

func (s *Suite) TestLocalnetCLIQueryTxAndWaitHappyPath() {
	cli := s.localnetCLI

	var balances banktypes.QueryAllBalancesResponse
	s.Require().NoError(cli.ExecQueryCmd(
		nibidcmd.QueryCmd(),
		[]string{"bank", "balances", nutil.LocalnetValAddr.String()},
		&balances,
	))
	s.Require().True(
		balances.Balances.AmountOf(appconst.DENOM_UNIBI).IsPositive(),
		"validator should have spendable localnet funds",
	)

	recipient := bytes.Repeat([]byte{0x42}, 20)
	txResp, err := cli.ExecTxCmd(
		nibidcmd.TxCmd(),
		[]string{"bank", "send", localnet.KeyName, sdk.AccAddress(recipient).String(), "1" + appconst.DENOM_UNIBI},
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(txResp.TxHash)

	nextHeight, err := cli.WaitForNextBlockVerbose()
	s.Require().NoError(err)
	s.Require().Positive(nextHeight)

	queriedTx, err := cli.QueryTx(txResp.TxHash)
	s.Require().NoError(err)
	s.Require().Equal(txResp.TxHash, queriedTx.TxHash)
}
