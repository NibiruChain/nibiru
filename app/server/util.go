package server

import (
	"net"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/net/netutil"

	srvconfig "github.com/NibiruChain/nibiru/v2/app/server/config"

	sdkserver "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version"

	tmcmd "github.com/NibiruChain/nibiru/v2/bft/cmd/nibiru-bft/commands"
	tmlog "github.com/NibiruChain/nibiru/v2/bft/libs/log"
	rpcclient "github.com/NibiruChain/nibiru/v2/bft/rpc/jsonrpc/client"
)

// AddCommands adds server commands
func AddCommands(
	rootCmd *cobra.Command,
	opts StartOptions,
	appExport types.AppExporter,
	addStartFlags types.ModuleInitFlags,
) {
	rootCmd.AddCommand(NodeCommands(opts, appExport, addStartFlags)...)
	rootCmd.AddCommand(version.NewVersionCommand())
}

// NodeCommands builds a fresh set of the commands for operating a node:
// start, tendermint (alias bft), export, and rollback. Each call returns new
// command objects, so the set can be attached to more than one parent.
func NodeCommands(
	opts StartOptions,
	appExport types.AppExporter,
	addStartFlags types.ModuleInitFlags,
) []*cobra.Command {
	tendermintCmd := &cobra.Command{
		Use:     "tendermint",
		Aliases: []string{"bft"},
		Short:   "Nibiru BFT consensus commands",
	}

	tendermintCmd.AddCommand(
		sdkserver.ShowNodeIDCmd(),
		sdkserver.ShowValidatorCmd(),
		sdkserver.ShowAddressCmd(),
		sdkserver.VersionCmd(),
		tmcmd.NewResetAllCmd(),
		tmcmd.NewResetStateCmd(),
		sdkserver.BootstrapStateCmd(opts.AppCreator),
	)

	startCmd := StartCmd(opts)
	addStartFlags(startCmd)

	return []*cobra.Command{
		startCmd,
		tendermintCmd,
		sdkserver.ExportCmd(appExport, opts.DefaultNodeHome),
		sdkserver.NewRollbackCmd(opts.AppCreator, opts.DefaultNodeHome),
	}
}

func ConnectTmWS(tmRPCAddr, tmEndpoint string, logger tmlog.Logger) *rpcclient.WSClient {
	tmWsClient, err := rpcclient.NewWS(tmRPCAddr, tmEndpoint,
		rpcclient.MaxReconnectAttempts(256),
		rpcclient.ReadWait(120*time.Second),
		rpcclient.WriteWait(120*time.Second),
		rpcclient.PingPeriod(50*time.Second),
		rpcclient.OnReconnect(func() {
			logger.Debug("EVM RPC reconnects to Tendermint WS", "address", tmRPCAddr+tmEndpoint)
		}),
	)

	if err != nil {
		logger.Error(
			"Tendermint WS rpcClient could not be created",
			"address", tmRPCAddr+tmEndpoint,
			"error", err,
		)
	} else if err := tmWsClient.OnStart(); err != nil {
		logger.Error(
			"Tendermint WS rpcClient could not start",
			"address", tmRPCAddr+tmEndpoint,
			"error", err,
		)
	}

	return tmWsClient
}

// Listen starts a net.Listener on the tcp network on the given address.
// If there is a specified MaxOpenConnections in the config, it will also set the limitListener.
func Listen(addr string, config *srvconfig.Config) (net.Listener, error) {
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	if config.JSONRPC.MaxOpenConnections > 0 {
		ln = netutil.LimitListener(ln, config.JSONRPC.MaxOpenConnections)
	}
	return ln, err
}
