package testnetwork_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testnetwork"
)

func Test(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	suite.Suite
}

func (s *Suite) TestLocalnetCLIRenderQueryCmd() {
	localnetCLI := testnetwork.LocalnetCLI{
		FromName: testnetwork.LocalnetKeyName,
		FromAddr: nutil.LocalnetValAddr,
		NodeURI:  testnetwork.LocalnetNodeURI,
		TxFee:    testnetwork.LocalnetTxFee,
		TxGas:    testnetwork.LocalnetTxGas,
	}

	cmd := &cobra.Command{Use: "tokenfactory"}
	got := localnetCLI.RenderQueryCmd(cmd, []string{"denom-info", "factory/nibi1abc/sub denom"})

	s.Require().Equal(
		"nibid q tokenfactory denom-info 'factory/nibi1abc/sub denom' --output=json --node=http://localhost:26657",
		got,
	)
}

func (s *Suite) TestLocalnetCLIRenderTxCmd() {
	localnetCLI := testnetwork.LocalnetCLI{
		FromName: testnetwork.LocalnetKeyName,
		FromAddr: nutil.LocalnetValAddr,
		NodeURI:  testnetwork.LocalnetNodeURI,
		TxFee:    testnetwork.LocalnetTxFee,
		TxGas:    testnetwork.LocalnetTxGas,
	}

	cmd := &cobra.Command{Use: "tokenfactory"}
	got := localnetCLI.RenderTxCmd(cmd, []string{"create-denom", "sub denom"})

	s.Require().Equal(
		"nibid tx tokenfactory create-denom 'sub denom' --from=validator --fees=1000unibi --gas=5000000 --yes=true --broadcast-mode=sync --chain-id=nibiru-localnet-0 --keyring-backend=test --node=http://localhost:26657 --output=json",
		got,
	)
}

func (s *Suite) TestLocalnetCLIRenderTxCmdWithOptions() {
	localnetCLI := testnetwork.LocalnetCLI{
		FromName: testnetwork.LocalnetKeyName,
		FromAddr: nutil.LocalnetValAddr,
		NodeURI:  testnetwork.LocalnetNodeURI,
		TxFee:    testnetwork.LocalnetTxFee,
		TxGas:    testnetwork.LocalnetTxGas,
	}

	cmd := &cobra.Command{Use: "wasm"}
	got := localnetCLI.RenderTxCmd(
		cmd,
		[]string{"store", "contract.wasm"},
		testnetwork.WithLocalnetTxGas("auto"),
		testnetwork.WithLocalnetTxGasAdjustment("1.5"),
		testnetwork.WithLocalnetTxFees("10000000unibi"),
	)

	s.Require().Equal(
		"nibid tx wasm store contract.wasm --from=validator --fees=10000000unibi --gas=auto --yes=true --broadcast-mode=sync --chain-id=nibiru-localnet-0 --keyring-backend=test --node=http://localhost:26657 --output=json --gas-adjustment=1.5",
		got,
	)
}
