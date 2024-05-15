package cmd

import (
	"errors"
	"io"
	"os"

	"github.com/NibiruChain/nibiru/app/server"
	srvconfig "github.com/NibiruChain/nibiru/app/server/config"

	"github.com/NibiruChain/nibiru/app/appconst"
	"github.com/NibiruChain/nibiru/x/sudo/cli"

	dbm "github.com/cometbft/cometbft-db"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/app"
	oraclecli "github.com/NibiruChain/nibiru/x/oracle/client/cli"
)

// NewRootCmd creates a new root command for nibid. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, app.EncodingConfig) {
	encodingConfig := app.MakeEncodingConfig()
	app.SetPrefixes(appconst.AccountAddressPrefix)

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("") // In simapp, we don't use any prefix for env variables.

	rootCmd := &cobra.Command{
		Use:     "nibid",
		Short:   "Nibiru blockchain node CLI",
		Aliases: []string{"nibiru"},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx, err := client.ReadPersistentCommandFlags(
				initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(
				initClientCtx, cmd,
			); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := srvconfig.AppConfig("unibi")
			tmCfg := customTendermintConfig()

			return sdkserver.InterceptConfigsPreRunHandler(
				cmd,
				customAppTemplate,
				customAppConfig,
				tmCfg,
			)
		},
	}

	initRootCmd(rootCmd, encodingConfig)

	return rootCmd, encodingConfig
}

/*
'initRootCmd' adds keybase, auxiliary RPC, query, and transaction (tx) child
commands, then builds the rosetta root command given a protocol buffers
serializer/deserializer.

Args:

	rootCmd: The root command called once in the 'main.go' of 'nibid'.
	encodingConfig: EncodingConfig specifies the concrete encoding types to use
	  for a given app. This is provided for compatibility between protobuf and
	  amino implementations.
*/
func initRootCmd(rootCmd *cobra.Command, encodingConfig app.EncodingConfig) {
	a := appCreator{encodingConfig}
	rootCmd.AddCommand(
		InitCmd(app.ModuleBasics, app.DefaultNodeHome),
		AddGenesisAccountCmd(app.DefaultNodeHome),
		GetBuildWasmMsg(),
		DecodeBase64Cmd(app.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		testnetCmd(app.ModuleBasics, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),
		config.Cmd(),
		pruning.PruningCmd(a.newApp),
	)

	server.AddCommands(
		rootCmd,
		server.NewDefaultStartOptions(a.newApp, app.DefaultNodeHome),
		a.appExport,
		addModuleInitFlags,
	)

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		rpc.StatusCommand(),
		genesisCommand(
			encodingConfig,
			oraclecli.AddGenesisPricefeederDelegationCmd(app.DefaultNodeHome),
			cli.AddSudoRootAccountCmd(app.DefaultNodeHome),
		),
		queryCommand(),
		txCommand(),
		keys.Commands(app.DefaultNodeHome),
	)

	// TODO add rosettaj
	// add rosetta
	// rootCmd.AddCommand(
	//	server.RosettaCommand(
	//		encodingConfig.InterfaceRegistry, encodingConfig.Codec))
}

// Implements the servertypes.ModuleInitFlags interface
func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
}

// genesisCommand builds genesis-related `simd genesis` command. Users may provide application specific commands as a parameter
func genesisCommand(encodingConfig app.EncodingConfig, cmds ...*cobra.Command) *cobra.Command {
	cmd := genutilcli.GenesisCoreCommand(encodingConfig.TxConfig, app.ModuleBasics, app.DefaultNodeHome)

	for _, subCmd := range cmds {
		cmd.AddCommand(subCmd)
	}
	return cmd
}

func queryCommand() *cobra.Command {
	rootQueryCmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	rootQueryCmd.AddCommand(
		authcmd.GetAccountCmd(),
		rpc.ValidatorCommand(),
		rpc.BlockCommand(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
	)

	// Adds all query commands to the 'rootQueryCmd'
	app.ModuleBasics.AddQueryCommands(rootQueryCmd)
	rootQueryCmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return rootQueryCmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
	)

	app.ModuleBasics.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

type appCreator struct {
	encCfg app.EncodingConfig
}

// newApp is an appCreator
func (a appCreator) newApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	baseappOptions := sdkserver.DefaultBaseappOptions(appOpts)

	return app.NewNibiruApp(
		logger, db, traceStore, true,
		a.encCfg,
		appOpts,
		baseappOptions...,
	)
}

// appExport creates a new simapp (optionally at a given height)
// and exports state.
func (a appCreator) appExport(
	logger log.Logger, db dbm.DB, traceStore io.Writer, height int64,
	forZeroHeight bool, jailAllowedAddrs []string, appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var nibiruApp *app.NibiruApp
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home is not set")
	}

	loadLatestHeight := height == -1
	nibiruApp = app.NewNibiruApp(
		logger,
		db,
		traceStore,
		loadLatestHeight,
		a.encCfg,
		appOpts,
	)
	if height != -1 {
		if err := nibiruApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return nibiruApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}
