package localnet_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/localnet"
)

func Test(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	suite.Suite
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
	if err := nutil.EnsureLocalBlockchain(); err != nil {
		s.T().Skipf("localnet unavailable: %v", err)
	}

	cli, err := localnet.NewCLI()
	s.Require().NoError(err)
	s.Require().NotNil(cli.EvmRpc.Eth)
	s.Require().NotNil(cli.EvmRpc.Filters)
	s.Require().NotNil(cli.EvmRpc.Net)
	s.Require().NotNil(cli.EvmRpc.Debug)
}
