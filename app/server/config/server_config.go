// Copyright (c) 2023-2024 Nibi, Inc.
package config

import (
	"errors"
	"fmt"
	"path"
	"time"

	tracerslogger "github.com/ethereum/go-ethereum/eth/tracers/logger"

	"github.com/spf13/viper"

	"github.com/cometbft/cometbft/libs/strings"

	sdkioerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/server/config"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	// DefaultAPIEnable is the default value for the parameter that defines if the cosmos REST API server is enabled
	DefaultAPIEnable = false

	// DefaultGRPCEnable is the default value for the parameter that defines if the gRPC server is enabled
	DefaultGRPCEnable = false

	// DefaultGRPCWebEnable is the default value for the parameter that defines if the gRPC web server is enabled
	DefaultGRPCWebEnable = false

	// DefaultJSONRPCEnable is the default value for the parameter that defines if the JSON-RPC server is enabled
	DefaultJSONRPCEnable = false

	// DefaultRosettaEnable is the default value for the parameter that defines if the Rosetta API server is enabled
	DefaultRosettaEnable = false

	// DefaultTelemetryEnable is the default value for the parameter that defines if the telemetry is enabled
	DefaultTelemetryEnable = false

	// DefaultGRPCAddress is the default address the gRPC server binds to.
	DefaultGRPCAddress = "0.0.0.0:9900"

	// DefaultJSONRPCAddress is the default address the JSON-RPC server binds to.
	DefaultJSONRPCAddress = "127.0.0.1:8545"

	// DefaultJSONRPCWsAddress is the default address the JSON-RPC WebSocket server binds to.
	DefaultJSONRPCWsAddress = "127.0.0.1:8546"

	// DefaultJsonRPCMetricsAddress is the default address the JSON-RPC Metrics server binds to.
	DefaultJSONRPCMetricsAddress = "127.0.0.1:6065"

	// DefaultEVMTracer is the default vm.Tracer type
	DefaultEVMTracer = ""

	// DefaultFixRevertGasRefundHeight is the default height at which to overwrite gas refund
	DefaultFixRevertGasRefundHeight = 0

	// DefaultMaxTxGasWanted is the default gas wanted for each eth tx returned in ante handler in check tx mode
	DefaultMaxTxGasWanted = 0

	// DefaultEthCallGasLimit is the default cap on gas that can be used in eth_call/estimateGas
	DefaultEthCallGasLimit uint64 = 25_000_000

	// DefaultFilterCap is the default cap for total number of filters that can be created
	DefaultFilterCap int32 = 200

	// DefaultFeeHistoryCap is the default cap for total number of blocks that can be fetched
	DefaultFeeHistoryCap int32 = 100

	// DefaultLogsCap is the default cap of results returned from single 'eth_getLogs' query
	DefaultLogsCap int32 = 10000

	// DefaultBlockRangeCap is the default cap of block range allowed for 'eth_getLogs' query
	DefaultBlockRangeCap int32 = 10000

	// DefaultEVMTimeout is the default timeout for eth_call
	DefaultEVMTimeout = 5 * time.Second

	// DefaultTxFeeCap is the default tx-fee cap for sending a transaction
	DefaultTxFeeCap float64 = 1.0

	// DefaultHTTPTimeout is the default read/write timeout of the http json-rpc server
	DefaultHTTPTimeout = 30 * time.Second

	// DefaultHTTPIdleTimeout is the default idle timeout of the http json-rpc server
	DefaultHTTPIdleTimeout = 120 * time.Second

	// DefaultAllowUnprotectedTxs value is false
	DefaultAllowUnprotectedTxs = false

	// DefaultMaxOpenConnections represents the amount of open connections (unlimited = 0)
	DefaultMaxOpenConnections = 0

	// DefaultGasAdjustment value to use as default in gas-adjustment flag
	DefaultGasAdjustment = 1.2

	// DefaultZeroCopy is the default value that defines if
	// the zero-copied slices must be retained beyond current block's execution
	// the sdk address cache will be disabled if zero-copy is enabled
	DefaultZeroCopy = false
)

var evmTracers = []string{"json", "markdown", "struct", "access_list"}

// Config defines the server's top level configuration. It includes the default app config
// from the SDK as well as the EVM configuration to enable the JSON-RPC APIs.
type Config struct {
	config.Config `mapstructure:",squash"`

	EVM     EVMConfig     `mapstructure:"evm"`
	JSONRPC JSONRPCConfig `mapstructure:"json-rpc"`
	TLS     TLSConfig     `mapstructure:"tls"`
}

// EVMConfig defines the application configuration values for the EVM.
type EVMConfig struct {
	// Tracer defines vm.Tracer type that the EVM will use if the node is run in
	// trace mode. Default: 'json'.
	Tracer     string               `mapstructure:"tracer"`
	TracerOpts tracerslogger.Config `mapstucture:"tracer_opts"`
	// MaxTxGasWanted defines the gas wanted for each eth tx returned in ante handler in check tx mode.
	MaxTxGasWanted uint64 `mapstructure:"max-tx-gas-wanted"`
}

// JSONRPCConfig defines configuration for the EVM RPC server.
type JSONRPCConfig struct {
	// API defines a list of JSON-RPC namespaces that should be enabled
	API []string `mapstructure:"api"`
	// Address defines the HTTP server to listen on
	Address string `mapstructure:"address"`
	// WsAddress defines the WebSocket server to listen on
	WsAddress string `mapstructure:"ws-address"`
	// GasCap is the global gas cap for eth-call variants.
	GasCap uint64 `mapstructure:"gas-cap"`
	// EVMTimeout is the global timeout for eth-call.
	EVMTimeout time.Duration `mapstructure:"evm-timeout"`
	// TxFeeCap is the global tx-fee cap for send transaction
	TxFeeCap float64 `mapstructure:"txfee-cap"`
	// FilterCap is the global cap for total number of filters that can be created.
	FilterCap int32 `mapstructure:"filter-cap"`
	// FeeHistoryCap is the global cap for total number of blocks that can be fetched
	FeeHistoryCap int32 `mapstructure:"feehistory-cap"`
	// Enable defines if the EVM RPC server should be enabled.
	Enable bool `mapstructure:"enable"`
	// LogsCap defines the max number of results can be returned from single `eth_getLogs` query.
	LogsCap int32 `mapstructure:"logs-cap"`
	// BlockRangeCap defines the max block range allowed for `eth_getLogs` query.
	BlockRangeCap int32 `mapstructure:"block-range-cap"`
	// HTTPTimeout is the read/write timeout of http json-rpc server.
	HTTPTimeout time.Duration `mapstructure:"http-timeout"`
	// HTTPIdleTimeout is the idle timeout of http json-rpc server.
	HTTPIdleTimeout time.Duration `mapstructure:"http-idle-timeout"`
	// AllowUnprotectedTxs restricts unprotected (non EIP155 signed) transactions to be submitted via
	// the node's RPC when global parameter is disabled.
	AllowUnprotectedTxs bool `mapstructure:"allow-unprotected-txs"`
	// MaxOpenConnections sets the maximum number of simultaneous connections
	// for the server listener.
	MaxOpenConnections int `mapstructure:"max-open-connections"`
	// EnableIndexer defines if enable the custom indexer service.
	EnableIndexer bool `mapstructure:"enable-indexer"`
	// MetricsAddress defines the metrics server to listen on
	MetricsAddress string `mapstructure:"metrics-address"`
	// FixRevertGasRefundHeight defines the upgrade height for fix of revert gas refund logic when transaction reverted
	FixRevertGasRefundHeight int64 `mapstructure:"fix-revert-gas-refund-height"`
}

// TLSConfig defines the certificate and matching private key for the server.
type TLSConfig struct {
	// CertificatePath the file path for the certificate .pem file
	CertificatePath string `mapstructure:"certificate-path"`
	// KeyPath the file path for the key .pem file
	KeyPath string `mapstructure:"key-path"`
}

// AppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func AppConfig(denom string) (string, any) {
	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	customAppConfig := DefaultConfig()

	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	if denom != "" {
		customAppConfig.MinGasPrices = "0" + denom
	}

	customAppTemplate := config.DefaultConfigTemplate + DefaultConfigTemplate

	return customAppTemplate, *customAppConfig
}

// DefaultConfig returns server's default configuration.
func DefaultConfig() *Config {
	defaultSDKConfig := config.DefaultConfig()
	defaultSDKConfig.API.Enable = DefaultAPIEnable
	defaultSDKConfig.GRPC.Enable = DefaultGRPCEnable
	defaultSDKConfig.GRPCWeb.Enable = DefaultGRPCWebEnable
	defaultSDKConfig.Rosetta.Enable = DefaultRosettaEnable
	defaultSDKConfig.Telemetry.Enabled = DefaultTelemetryEnable

	return &Config{
		Config:  *defaultSDKConfig,
		EVM:     *DefaultEVMConfig(),
		JSONRPC: *DefaultJSONRPCConfig(),
		TLS:     *DefaultTLSConfig(),
	}
}

// DefaultEVMConfig returns the default EVM configuration
func DefaultEVMConfig() *EVMConfig {
	return &EVMConfig{
		Tracer: DefaultEVMTracer,
		TracerOpts: tracerslogger.Config{
			EnableMemory:     false, // disable
			DisableStack:     false, // enable stack
			DisableStorage:   false, // enable storage
			EnableReturnData: false, // disable
			Debug:            true,  // enable debug
			Limit:            0,
			Overrides:        nil,
		},
		MaxTxGasWanted: DefaultMaxTxGasWanted,
	}
}

// Validate returns an error if the tracer type is invalid.
func (c EVMConfig) Validate() error {
	if c.Tracer != "" && !strings.StringInSlice(c.Tracer, evmTracers) {
		return fmt.Errorf("invalid tracer type %s, available types: %v", c.Tracer, evmTracers)
	}

	return nil
}

// GetDefaultAPINamespaces returns the default list of JSON-RPC namespaces that should be enabled
func GetDefaultAPINamespaces() []string {
	return []string{"eth", "net", "web3"}
}

// GetAPINamespaces returns the all the available JSON-RPC API namespaces.
func GetAPINamespaces() []string {
	return []string{"web3", "eth", "net", "txpool", "debug"}
}

// DefaultJSONRPCConfig returns an EVM config with the JSON-RPC API enabled by default
func DefaultJSONRPCConfig() *JSONRPCConfig {
	return &JSONRPCConfig{
		Enable:                   false,
		API:                      GetDefaultAPINamespaces(),
		Address:                  DefaultJSONRPCAddress,
		WsAddress:                DefaultJSONRPCWsAddress,
		GasCap:                   DefaultEthCallGasLimit,
		EVMTimeout:               DefaultEVMTimeout,
		TxFeeCap:                 DefaultTxFeeCap,
		FilterCap:                DefaultFilterCap,
		FeeHistoryCap:            DefaultFeeHistoryCap,
		BlockRangeCap:            DefaultBlockRangeCap,
		LogsCap:                  DefaultLogsCap,
		HTTPTimeout:              DefaultHTTPTimeout,
		HTTPIdleTimeout:          DefaultHTTPIdleTimeout,
		AllowUnprotectedTxs:      DefaultAllowUnprotectedTxs,
		MaxOpenConnections:       DefaultMaxOpenConnections,
		EnableIndexer:            false,
		MetricsAddress:           DefaultJSONRPCMetricsAddress,
		FixRevertGasRefundHeight: DefaultFixRevertGasRefundHeight,
	}
}

// Validate returns an error if the JSON-RPC configuration fields are invalid.
func (c JSONRPCConfig) Validate() error {
	if c.Enable && len(c.API) == 0 {
		return errors.New("cannot enable JSON-RPC without defining any API namespace")
	}

	if c.FilterCap < 0 {
		return errors.New("JSON-RPC filter-cap cannot be negative")
	}

	if c.FeeHistoryCap <= 0 {
		return errors.New("JSON-RPC feehistory-cap cannot be negative or 0")
	}

	if c.TxFeeCap < 0 {
		return errors.New("JSON-RPC tx fee cap cannot be negative")
	}

	if c.EVMTimeout < 0 {
		return errors.New("JSON-RPC EVM timeout duration cannot be negative")
	}

	if c.LogsCap < 0 {
		return errors.New("JSON-RPC logs cap cannot be negative")
	}

	if c.BlockRangeCap < 0 {
		return errors.New("JSON-RPC block range cap cannot be negative")
	}

	if c.HTTPTimeout < 0 {
		return errors.New("JSON-RPC HTTP timeout duration cannot be negative")
	}

	if c.HTTPIdleTimeout < 0 {
		return errors.New("JSON-RPC HTTP idle timeout duration cannot be negative")
	}

	// check for duplicates
	seenAPIs := make(map[string]bool)
	for _, api := range c.API {
		if seenAPIs[api] {
			return fmt.Errorf("repeated API namespace '%s'", api)
		}

		seenAPIs[api] = true
	}

	return nil
}

// DefaultTLSConfig returns the default TLS configuration
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		CertificatePath: "",
		KeyPath:         "",
	}
}

// Validate returns an error if the TLS certificate and key file extensions are invalid.
func (c TLSConfig) Validate() error {
	certExt := path.Ext(c.CertificatePath)

	if c.CertificatePath != "" && certExt != ".pem" {
		return fmt.Errorf("invalid extension %s for certificate path %s, expected '.pem'", certExt, c.CertificatePath)
	}

	keyExt := path.Ext(c.KeyPath)

	if c.KeyPath != "" && keyExt != ".pem" {
		return fmt.Errorf("invalid extension %s for key path %s, expected '.pem'", keyExt, c.KeyPath)
	}

	return nil
}

// GetConfig returns a fully parsed Config object.
func GetConfig(v *viper.Viper) (Config, error) {
	conf := DefaultConfig()
	if err := v.Unmarshal(conf); err != nil {
		return Config{}, fmt.Errorf("error extracting app config: %w", err)
	}
	return *conf, nil
}

// ValidateBasic returns an error any of the application configuration fields are invalid
func (c Config) ValidateBasic() error {
	if err := c.EVM.Validate(); err != nil {
		return sdkioerrors.Wrapf(sdkerrors.ErrAppConfig, "invalid evm config value: %s", err.Error())
	}

	if err := c.JSONRPC.Validate(); err != nil {
		return sdkioerrors.Wrapf(sdkerrors.ErrAppConfig, "invalid json-rpc config value: %s", err.Error())
	}

	if err := c.TLS.Validate(); err != nil {
		return sdkioerrors.Wrapf(sdkerrors.ErrAppConfig, "invalid tls config value: %s", err.Error())
	}

	return c.Config.ValidateBasic()
}

// DefaultConfigTemplate defines the configuration template for the EVM RPC configuration
const DefaultConfigTemplate = `
###############################################################################
###                             EVM Configuration                           ###
###############################################################################

[evm]

# Tracer defines the 'vm.Tracer' type that the EVM will use when the node is run in
# debug mode. To enable tracing use the '--evm.tracer' flag when starting your node.
# Valid types are: json|struct|access_list|markdown
tracer = "{{ .EVM.Tracer }}"

# MaxTxGasWanted defines the gas wanted for each eth tx returned in ante handler in check tx mode.
max-tx-gas-wanted = {{ .EVM.MaxTxGasWanted }}

[evm.tracer_opts]

# Enable the capture of EVM memory state at each
# execution step. This can be useful for debugging complex contracts but may
# significantly increase the volume of logged data.
memory = false

# Enable the capture of contract storage changes. By default, storage
# modifications are logged. Disabling storage capture can significantly reduce
# log size for contracts with many storage operations.
stack = true

# Enable the capture of contract storage changes.
storage = true 

# enable return-data capture
return-data = false

# enable debug capture
debug = true

# Maximum length of the tracer output. Zero means unlimited.
limit = 0

###############################################################################
###                           JSON RPC Configuration                        ###
###############################################################################

[json-rpc]

# Enable defines if the gRPC server should be enabled.
enable = {{ .JSONRPC.Enable }}

# Address defines the EVM RPC HTTP server address to bind to.
address = "{{ .JSONRPC.Address }}"

# Address defines the EVM WebSocket server address to bind to.
ws-address = "{{ .JSONRPC.WsAddress }}"

# API defines a list of JSON-RPC namespaces that should be enabled
# Example: "eth,txpool,net,debug,web3"
api = "{{range $index, $elmt := .JSONRPC.API}}{{if $index}},{{$elmt}}{{else}}{{$elmt}}{{end}}{{end}}"

# GasCap sets a cap on gas that can be used in eth_call/estimateGas (0=infinite). Default: 25,000,000.
gas-cap = {{ .JSONRPC.GasCap }}

# EVMTimeout is the global timeout for eth_call. Default: 5s.
evm-timeout = "{{ .JSONRPC.EVMTimeout }}"

# TxFeeCap is the global tx-fee cap for send transaction. Default: 1eth.
txfee-cap = {{ .JSONRPC.TxFeeCap }}

# FilterCap sets the global cap for total number of filters that can be created
filter-cap = {{ .JSONRPC.FilterCap }}

# FeeHistoryCap sets the global cap for total number of blocks that can be fetched
feehistory-cap = {{ .JSONRPC.FeeHistoryCap }}

# LogsCap defines the max number of results can be returned from single 'eth_getLogs' query.
logs-cap = {{ .JSONRPC.LogsCap }}

# BlockRangeCap defines the max block range allowed for 'eth_getLogs' query.
block-range-cap = {{ .JSONRPC.BlockRangeCap }}

# HTTPTimeout is the read/write timeout of http json-rpc server.
http-timeout = "{{ .JSONRPC.HTTPTimeout }}"

# HTTPIdleTimeout is the idle timeout of http json-rpc server.
http-idle-timeout = "{{ .JSONRPC.HTTPIdleTimeout }}"

# AllowUnprotectedTxs restricts unprotected (non EIP155 signed) transactions to be submitted via
# the node's RPC when the global parameter is disabled.
allow-unprotected-txs = {{ .JSONRPC.AllowUnprotectedTxs }}

# MaxOpenConnections sets the maximum number of simultaneous connections
# for the server listener.
max-open-connections = {{ .JSONRPC.MaxOpenConnections }}

# EnableIndexer enables the custom transaction indexer for the EVM (ethereum transactions).
enable-indexer = {{ .JSONRPC.EnableIndexer }}

# MetricsAddress defines the EVM Metrics server address to bind to. Pass --metrics in CLI to enable
# Prometheus metrics path: /debug/metrics/prometheus
metrics-address = "{{ .JSONRPC.MetricsAddress }}"

# Upgrade height for fix of revert gas refund logic when transaction reverted.
fix-revert-gas-refund-height = {{ .JSONRPC.FixRevertGasRefundHeight }}

###############################################################################
###                             TLS Configuration                           ###
###############################################################################

[tls]

# Certificate path defines the cert.pem file path for the TLS configuration.
certificate-path = "{{ .TLS.CertificatePath }}"

# Key path defines the key.pem file path for the TLS configuration.
key-path = "{{ .TLS.KeyPath }}"
`
