package server

import (
	"context"
	"net"
	"time"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/spf13/cobra"
	"golang.org/x/net/netutil"

	srvconfig "github.com/NibiruChain/nibiru/app/server/config"

	"github.com/cosmos/cosmos-sdk/client"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/version"

	"cosmossdk.io/log"
	tmcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	rpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
)

// AddCommands adds server commands
func AddCommands(
	rootCmd *cobra.Command,
	opts StartOptions,
	appExport types.AppExporter,
	addStartFlags types.ModuleInitFlags,
) {
	tendermintCmd := &cobra.Command{
		Use:   "tendermint",
		Short: "Tendermint subcommands",
	}

	tendermintCmd.AddCommand(
		sdkserver.ShowNodeIDCmd(),
		sdkserver.ShowValidatorCmd(),
		sdkserver.ShowAddressCmd(),
		sdkserver.VersionCmd(),
		tmcmd.ResetAllCmd,
		tmcmd.ResetStateCmd,
		sdkserver.BootstrapStateCmd(opts.AppCreator),
	)

	startCmd := StartCmd(opts.AppCreator, opts.DefaultNodeHome)
	addStartFlags(startCmd)

	rootCmd.AddCommand(
		startCmd,
		tendermintCmd,
		sdkserver.ExportCmd(appExport, opts.DefaultNodeHome),
		version.NewVersionCommand(),
		sdkserver.NewRollbackCmd(opts.AppCreator, opts.DefaultNodeHome),

		// custom tx indexer command
		//NewIndexTxCmd(), TODO: check indexer tx command
	)
}

// StatusCommand returns the command to return the status of the network.
func StatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Query remote node for status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			status, err := cmtservice.GetNodeStatus(context.Background(), clientCtx)
			if err != nil {
				return err
			}

			output, err := cmtjson.Marshal(status)
			if err != nil {
				return err
			}

			// In order to maintain backwards compatibility, the default json format output
			outputFormat, _ := cmd.Flags().GetString(flags.FlagOutput)
			if outputFormat == flags.OutputFormatJSON {
				clientCtx = clientCtx.WithOutputFormat(flags.OutputFormatJSON)
			}

			return clientCtx.PrintRaw(output)
		},
	}

	cmd.Flags().StringP(flags.FlagNode, "n", "tcp://localhost:26657", "Node to connect to")
	cmd.Flags().StringP(flags.FlagOutput, "o", "json", "Output format (text|json)")

	return cmd
}

func ConnectTmWS(tmRPCAddr, tmEndpoint string, logger log.Logger) *rpcclient.WSClient {
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
			"Tendermint WS client could not be created",
			"address", tmRPCAddr+tmEndpoint,
			"error", err,
		)
	} else if err := tmWsClient.OnStart(); err != nil {
		logger.Error(
			"Tendermint WS client could not start",
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
