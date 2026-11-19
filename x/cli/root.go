package main

import (
	"errors"
	"io"
	"os"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	cmtcli "github.com/NibiruChain/nibiru/v2/bft/libs/cli"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/pruning"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/rpc"
	sdkserver "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	servertypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/types"
	simtestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/sims"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/crisis"
	genutilcli "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/genutil/client/cli"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/server"
	srvconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/x/cli/debugcmd"
	nibidcmd "github.com/NibiruChain/nibiru/v2/x/cli/impl"
	nutilflags "github.com/NibiruChain/nibiru/v2/x/nutil/flags"
	"github.com/NibiruChain/nibiru/v2/x/sudo/cli"
)

// NewRootCmd creates a new root command for nibid. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, app.EncodingConfig) {
	return NewRootCmdNamed(InvokedName(os.Args[0]))
}

// NewRootCmdNamed is NewRootCmd with an explicit command name, which help and
// usage text print. Use InvokedName to derive it from the process name.
func NewRootCmdNamed(name string) (*cobra.Command, app.EncodingConfig) {
	// we "pre"-instantiate the application for getting the injected/configured encoding configuration
	tmpAppDir, err := os.MkdirTemp("", "nibid-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpAppDir)
	tempApp := app.NewNibiruApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(tmpAppDir))
	encodingConfig := app.EncodingConfig{
		InterfaceRegistry: tempApp.InterfaceRegistry(),
		Codec:             tempApp.AppCodec(),
		TxConfig:          tempApp.GetTxConfig(),
		Amino:             tempApp.LegacyAmino(),
	}

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
		Use:   name,
		Short: "Nibiru CLI: send transactions, query the chain, and run a node.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// --help-verbose prints help for any command instead of running it.
			if show, _ := cmd.Flags().GetBool(nutilflags.FlagHelpVerbose); show {
				return pflag.ErrHelp
			}
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx, err := client.ReadPersistentCommandFlags(
				initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, queryMode, err := nibidcmd.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(
				initClientCtx, cmd,
			); err != nil {
				return err
			}
			nibidcmd.SetQueryMode(cmd, queryMode)

			customAppTemplate, customAppConfig := srvconfig.AppConfig("unibi")
			tmCfg := appconst.NewDefaultTendermintConfig()

			return sdkserver.InterceptConfigsPreRunHandler(
				cmd,
				customAppTemplate,
				customAppConfig,
				tmCfg,
			)
		},
	}

	initRootCmd(rootCmd, encodingConfig)
	nutilflags.AddHelpVerboseToRoot(rootCmd)
	setRootUsage(rootCmd)
	followInvokedName(rootCmd, name)

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
	rootCmd.AddCommand(
		GetBuildWasmMsg(),
		DecodeBase64Cmd(app.DefaultNodeHome),
		LocalnetCmd(),
		cmtcli.NewCompletionCmd(rootCmd, true),
		debugcmd.Cmd(DecodeBase64Any),
		nibidcmd.ConfigCmd(),
		version.NewVersionCommand(),

		// add keybase, auxiliary RPC, query, and tx child commands
		rpc.StatusCommand(),
		nibidcmd.QueryCmd(),
		nibidcmd.TxCmd(),
		keysCommand(app.DefaultNodeHome),
	)

	addNodeCommands(rootCmd, func() []*cobra.Command {
		return newNodeCommands(encodingConfig)
	})

	setRootRowShorts(rootCmd)

	// TODO add rosettaj
	// add rosetta
	// rootCmd.AddCommand(
	//	server.RosettaCommand(
	//		encodingConfig.InterfaceRegistry, encodingConfig.Codec))
}

// newNodeCommands builds a fresh copy of every command for operating a node.
// Each call returns new command objects because a cobra command has one
// parent: the set is attached under `nibid node` and, hidden, at the top level.
func newNodeCommands(encodingConfig app.EncodingConfig) []*cobra.Command {
	a := appCreator{}

	cmds := server.NodeCommands(
		server.NewDefaultStartOptions(a.newApp, app.DefaultNodeHome),
		a.appExport,
		addModuleInitFlags,
	)
	return append(cmds,
		InitCmd(app.ModuleBasics, app.DefaultNodeHome),
		testnetCmd(app.ModuleBasics, banktypes.GenesisBalancesIterator{}),
		pruning.Cmd(a.newApp, app.DefaultNodeHome),
		genesisCommand(
			encodingConfig,
			cli.AddSudoRootAccountCmd(app.DefaultNodeHome),
		),
		// EVM Tx Indexer force catch up command
		server.NewEVMTxIndexCmd(),
	)
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

type appCreator struct{}

// newApp is an appCreator
func (a appCreator) newApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	baseappOptions := sdkserver.DefaultBaseappOptions(appOpts)

	return app.NewNibiruApp(
		logger, db, traceStore, true,
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
		appOpts,
	)
	if height != -1 {
		if err := nibiruApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return nibiruApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}
