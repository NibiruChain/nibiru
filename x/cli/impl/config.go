package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
)

const (
	// QueryModeConfigKey is the client.toml key that selects the configured
	// query backend.
	QueryModeConfigKey = "query-mode"

	defaultKeyringBackend = "os"
	defaultOutput         = "text"
	defaultNode           = "tcp://localhost:26657"
	defaultBroadcastMode  = "sync"
)

// QueryMode selects the configured backend for eligible CLI read operations.
// The effective backend also depends on the chain ID and operation allowlist.
type QueryMode string

const (
	// QueryModeGQL selects the GraphQL-backed query mode for eligible reads.
	QueryModeGQL QueryMode = "gql"
	// QueryModeDirect keeps all reads on the direct RPC path.
	QueryModeDirect QueryMode = "direct"
)

type queryModeContextKey struct{}

type clientConfig struct {
	ChainID        string `mapstructure:"chain-id" json:"chain-id"`
	KeyringBackend string `mapstructure:"keyring-backend" json:"keyring-backend"`
	Output         string `mapstructure:"output" json:"output"`
	Node           string `mapstructure:"node" json:"node"`
	BroadcastMode  string `mapstructure:"broadcast-mode" json:"broadcast-mode"`
}

type clientConfigOutput struct {
	ChainID        string    `json:"chain-id"`
	KeyringBackend string    `json:"keyring-backend"`
	Output         string    `json:"output"`
	Node           string    `json:"node"`
	BroadcastMode  string    `json:"broadcast-mode"`
	QueryMode      QueryMode `json:"query-mode"`
}

type clientConfigFile struct {
	path   string
	values map[string]any
}

type configWriter func(clientConfigFile) error

// ConfigCmd returns the Nibiru-owned command for reading and updating client
// configuration values in client.toml.
func ConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config <key> [value]",
		Short: "Create or query the Nibiru CLI configuration file",
		Args:  cobra.RangeArgs(0, 2),
		RunE:  runConfigCmd,
	}
}

// ReadFromClientConfig loads client.toml, repairs a missing or invalid
// query-mode value to gql, and initializes the Cosmos client context.
func ReadFromClientConfig(ctx client.Context) (client.Context, QueryMode, error) {
	if ctx.Viper == nil {
		ctx = ctx.WithViper("")
	}

	conf, queryMode, _, err := loadClientConfig(ctx)
	if err != nil {
		return ctx, QueryModeGQL, err
	}

	ctx = ctx.WithOutputFormat(conf.Output).
		WithChainID(conf.ChainID).
		WithKeyringDir(ctx.HomeDir)

	keyring, err := client.NewKeyringFromBackend(ctx, conf.KeyringBackend)
	if err != nil {
		return ctx, QueryModeGQL, fmt.Errorf("couldn't get key ring: %w", err)
	}
	ctx = ctx.WithKeyring(keyring)

	rpcClient, err := client.NewClientFromNode(conf.Node)
	if err != nil {
		return ctx, QueryModeGQL, fmt.Errorf("couldn't get client from node URI: %w", err)
	}

	ctx = ctx.WithNodeURI(conf.Node).
		WithClient(rpcClient).
		WithBroadcastMode(conf.BroadcastMode)

	return ctx, queryMode, nil
}

// SetQueryMode records the validated configured mode on the executed Cobra
// command. Query adapters can retrieve it with QueryModeFromCmd.
func SetQueryMode(cmd *cobra.Command, mode QueryMode) {
	cmd.SetContext(context.WithValue(cmd.Context(), queryModeContextKey{}, mode))
}

// QueryModeFromCmd returns the configured query mode for cmd. Commands that
// did not run the root pre-run use the safe gql default.
func QueryModeFromCmd(cmd *cobra.Command) QueryMode {
	mode, ok := cmd.Context().Value(queryModeContextKey{}).(QueryMode)
	if !ok {
		return QueryModeGQL
	}
	return mode
}

func runConfigCmd(cmd *cobra.Command, args []string) error {
	clientCtx := client.GetClientContextFromCmd(cmd)
	if clientCtx.Viper == nil {
		clientCtx = clientCtx.WithViper("")
	}

	conf, queryMode, configFile, err := loadClientConfig(clientCtx)
	if err != nil {
		return err
	}

	switch len(args) {
	case 0:
		return printClientConfig(cmd, conf, queryMode)
	case 1:
		return printClientConfigValue(cmd, conf, queryMode, args[0])
	case 2:
		return setClientConfigValue(configFile, args[0], args[1])
	default:
		panic("cobra validated the number of config command arguments")
	}
}

func loadClientConfig(ctx client.Context) (clientConfig, QueryMode, clientConfigFile, error) {
	return loadClientConfigWithWriter(ctx, writeClientConfigFile)
}

func loadClientConfigWithWriter(
	ctx client.Context, writer configWriter,
) (clientConfig, QueryMode, clientConfigFile, error) {
	configPath := filepath.Join(ctx.HomeDir, "config")
	configFilePath := filepath.Join(configPath, "client.toml")
	configFile, err := readClientConfigFile(configFilePath, ctx.ChainID, writer)
	if err != nil {
		return clientConfig{}, QueryModeGQL, clientConfigFile{}, err
	}

	queryMode, shouldRepair := normalizePersistedQueryMode(configFile.values[QueryModeConfigKey])
	if shouldRepair {
		configFile.values[QueryModeConfigKey] = string(queryMode)
		_ = writer(configFile)
	}

	conf, err := readEffectiveClientConfig(ctx, configPath)
	if err != nil {
		return clientConfig{}, QueryModeGQL, clientConfigFile{}, err
	}

	return conf, queryMode, configFile, nil
}

func readClientConfigFile(
	configFilePath string, chainID string, writer configWriter,
) (clientConfigFile, error) {
	contents, err := os.ReadFile(configFilePath)
	if os.IsNotExist(err) {
		configFile := clientConfigFile{
			path:   configFilePath,
			values: defaultClientConfigValues(chainID),
		}
		if err := writer(configFile); err != nil {
			return clientConfigFile{}, fmt.Errorf("could not write client config to the file: %w", err)
		}
		return configFile, nil
	}
	if err != nil {
		return clientConfigFile{}, fmt.Errorf("couldn't read client config: %w", err)
	}

	values := make(map[string]any)
	if err := toml.Unmarshal(contents, &values); err != nil {
		return clientConfigFile{}, fmt.Errorf("couldn't parse client config: %w", err)
	}

	return clientConfigFile{path: configFilePath, values: values}, nil
}

func readEffectiveClientConfig(ctx client.Context, configPath string) (clientConfig, error) {
	v := ctx.Viper
	v.AddConfigPath(configPath)
	v.SetConfigName("client")
	v.SetConfigType("toml")
	setClientConfigDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return clientConfig{}, err
	}

	var conf clientConfig
	if err := v.Unmarshal(&conf); err != nil {
		return clientConfig{}, err
	}
	return conf, nil
}

func setClientConfigDefaults(v interface{ SetDefault(string, any) }) {
	v.SetDefault(flags.FlagChainID, "")
	v.SetDefault(flags.FlagKeyringBackend, defaultKeyringBackend)
	v.SetDefault(flags.FlagOutput, defaultOutput)
	v.SetDefault(flags.FlagNode, defaultNode)
	v.SetDefault(flags.FlagBroadcastMode, defaultBroadcastMode)
}

func defaultClientConfigValues(chainID string) map[string]any {
	return map[string]any{
		flags.FlagChainID:        chainID,
		flags.FlagKeyringBackend: defaultKeyringBackend,
		flags.FlagOutput:         defaultOutput,
		flags.FlagNode:           defaultNode,
		flags.FlagBroadcastMode:  defaultBroadcastMode,
		QueryModeConfigKey:       string(QueryModeGQL),
	}
}

func normalizePersistedQueryMode(value any) (QueryMode, bool) {
	mode, ok := value.(string)
	if !ok {
		return QueryModeGQL, true
	}

	parsed, err := parseQueryMode(mode)
	if err != nil {
		return QueryModeGQL, true
	}
	return parsed, false
}

func parseQueryMode(value string) (QueryMode, error) {
	switch QueryMode(value) {
	case QueryModeGQL:
		return QueryModeGQL, nil
	case QueryModeDirect:
		return QueryModeDirect, nil
	default:
		return "", fmt.Errorf("invalid query mode %q: expected gql or direct", value)
	}
}

func printClientConfig(cmd *cobra.Command, conf clientConfig, queryMode QueryMode) error {
	output, err := json.MarshalIndent(clientConfigOutput{
		ChainID:        conf.ChainID,
		KeyringBackend: conf.KeyringBackend,
		Output:         conf.Output,
		Node:           conf.Node,
		BroadcastMode:  conf.BroadcastMode,
		QueryMode:      queryMode,
	}, "", "\t")
	if err != nil {
		return err
	}
	cmd.Println(string(output))
	return nil
}

func printClientConfigValue(
	cmd *cobra.Command, conf clientConfig, queryMode QueryMode, key string,
) error {
	switch key {
	case flags.FlagChainID:
		cmd.Println(conf.ChainID)
	case flags.FlagKeyringBackend:
		cmd.Println(conf.KeyringBackend)
	case flags.FlagOutput:
		cmd.Println(conf.Output)
	case flags.FlagNode:
		cmd.Println(conf.Node)
	case flags.FlagBroadcastMode:
		cmd.Println(conf.BroadcastMode)
	case QueryModeConfigKey:
		cmd.Println(queryMode)
	default:
		return errUnknownConfigKey(key)
	}
	return nil
}

func setClientConfigValue(configFile clientConfigFile, key, value string) error {
	switch key {
	case flags.FlagChainID, flags.FlagKeyringBackend, flags.FlagOutput, flags.FlagNode, flags.FlagBroadcastMode:
		configFile.values[key] = value
	case QueryModeConfigKey:
		mode, err := parseQueryMode(value)
		if err != nil {
			return err
		}
		configFile.values[key] = string(mode)
	default:
		return errUnknownConfigKey(key)
	}

	if err := writeClientConfigFile(configFile); err != nil {
		return fmt.Errorf("could not write client config to the file: %w", err)
	}
	return nil
}

func writeClientConfigFile(configFile clientConfigFile) error {
	contents, err := toml.Marshal(configFile.values)
	if err != nil {
		return err
	}

	configPath := filepath.Dir(configFile.path)
	if err := os.MkdirAll(configPath, 0o700); err != nil {
		return err
	}

	temporaryFile, err := os.CreateTemp(configPath, ".client.toml-*")
	if err != nil {
		return err
	}
	temporaryPath := temporaryFile.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	if err := temporaryFile.Chmod(0o600); err != nil {
		_ = temporaryFile.Close()
		return err
	}
	if _, err := temporaryFile.Write(contents); err != nil {
		_ = temporaryFile.Close()
		return err
	}
	if err := temporaryFile.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, configFile.path)
}

func errUnknownConfigKey(key string) error {
	return fmt.Errorf("unknown configuration key: %q", key)
}
