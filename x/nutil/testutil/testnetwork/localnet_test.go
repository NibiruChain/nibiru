package testnetwork

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/nutil"
)

func TestLocalnetCLIRenderQueryCmd(t *testing.T) {
	localnetCLI := LocalnetCLI{
		FromName: LocalnetKeyName,
		FromAddr: nutil.LocalnetValAddr,
		NodeURI:  LocalnetNodeURI,
		TxFee:    LocalnetTxFee,
		TxGas:    LocalnetTxGas,
	}

	cmd := &cobra.Command{Use: "tokenfactory"}
	got := localnetCLI.RenderQueryCmd(cmd, []string{"denom-info", "factory/nibi1abc/sub denom"})

	require.Equal(
		t,
		"nibid q tokenfactory denom-info 'factory/nibi1abc/sub denom' --output=json --node=http://localhost:26657",
		got,
	)
}

func TestLocalnetCLIRenderTxCmd(t *testing.T) {
	localnetCLI := LocalnetCLI{
		FromName: LocalnetKeyName,
		FromAddr: nutil.LocalnetValAddr,
		NodeURI:  LocalnetNodeURI,
		TxFee:    LocalnetTxFee,
		TxGas:    LocalnetTxGas,
	}

	cmd := &cobra.Command{Use: "tokenfactory"}
	got := localnetCLI.RenderTxCmd(cmd, []string{"create-denom", "sub denom"})

	require.Equal(
		t,
		"nibid tx tokenfactory create-denom 'sub denom' --from=validator --fees=1000unibi --gas=5000000 --yes=true --broadcast-mode=sync --chain-id=nibiru-localnet-0 --keyring-backend=test --node=http://localhost:26657 --output=json",
		got,
	)
}
