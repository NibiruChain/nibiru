package localnet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cometbft/cometbft/abci/types"
	cmtcli "github.com/cometbft/cometbft/libs/cli"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	rpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
)

const (
	ChainID        = "nibiru-localnet-0"
	KeyName        = "validator"
	NodeURI        = "http://localhost:26657"
	NodeWSURI      = "tcp://localhost:26657"
	NodeWSEndpoint = "/websocket"
	LocalnetEVMURI = "http://127.0.0.1:8545"
	TxFeeDefault   = "1000" + appconst.DENOM_UNIBI
	TxGasDefault   = "5000000"
)

type CLI struct {
	ClientCtx     client.Context
	FromName      string
	FromAddr      sdk.AccAddress
	NodeURI       string
	TxFee         string
	TxGas         string
	EthRpcBackend *rpcapi.Backend

	EvmRpcClient *ethclient.Client
	EvmRpc       EvmRpcAPI
}

type EvmRpcAPI struct {
	Eth     *rpcapi.EthAPI
	Net     *rpcapi.NetAPI
	Debug   *rpcapi.DebugAPI
	Filters *rpcapi.FiltersAPI
}

type TxOption func(*txOptions)

type txOptions struct {
	fromName      string
	txFee         string
	txGas         string
	gasAdjustment string
}

func WithTxFees(fees string) TxOption {
	return func(opts *txOptions) {
		opts.txFee = fees
	}
}

func WithTxGas(gas string) TxOption {
	return func(opts *txOptions) {
		opts.txGas = gas
	}
}

func WithTxGasAdjustment(gasAdjustment string) TxOption {
	return func(opts *txOptions) {
		opts.gasAdjustment = gasAdjustment
	}
}

func NewCLI() (CLI, error) {
	clientCtx, err := newClientCtx()
	if err != nil {
		return CLI{}, err
	}

	evmRpcClient, err := ethclient.Dial(LocalnetEVMURI)
	if err != nil {
		return CLI{}, fmt.Errorf("connect localnet EVM RPC client: %w", err)
	}

	tmWSClient, err := rpcclient.NewWS(NodeWSURI, NodeWSEndpoint)
	if err != nil {
		return CLI{}, fmt.Errorf("create localnet Tendermint websocket client: %w", err)
	}
	if err := tmWSClient.OnStart(); err != nil {
		return CLI{}, fmt.Errorf("start localnet Tendermint websocket client: %w", err)
	}

	serverCtx := server.NewDefaultContext()
	serverCtx.Logger = cmtlog.NewNopLogger()

	backend := rpcapi.NewBackend(serverCtx, serverCtx.Logger, clientCtx, false, nil)
	apis := rpcapi.GetRPCAPIs(
		serverCtx,
		clientCtx,
		tmWSClient,
		false,
		nil,
		[]string{
			rpcapi.NamespaceEth,
			rpcapi.NamespaceNet,
			rpcapi.NamespaceDebug,
			rpcapi.NamespaceWeb3,
			rpcapi.NamespaceTxPool,
		},
	)
	evmRpcAPI, err := buildEvmRpcAPI(apis)
	if err != nil {
		return CLI{}, err
	}

	return CLI{
		ClientCtx:     clientCtx,
		FromName:      KeyName,
		FromAddr:      nutil.LocalnetValAddr,
		NodeURI:       NodeURI,
		TxFee:         TxFeeDefault,
		TxGas:         TxGasDefault,
		EthRpcBackend: backend,
		EvmRpcClient:  evmRpcClient,
		EvmRpc:        evmRpcAPI,
	}, nil
}

func buildEvmRpcAPI(apis []gethrpc.API) (EvmRpcAPI, error) {
	var out EvmRpcAPI

	for _, api := range apis {
		switch svc := api.Service.(type) {
		case *rpcapi.EthAPI:
			out.Eth = svc
		case *rpcapi.FiltersAPI:
			out.Filters = svc
		case *rpcapi.NetAPI:
			out.Net = svc
		case *rpcapi.DebugAPI:
			out.Debug = svc
		}
	}

	if out.Eth == nil {
		return EvmRpcAPI{}, errors.New("localnet RPC APIs missing eth service")
	}
	if out.Filters == nil {
		return EvmRpcAPI{}, errors.New("localnet RPC APIs missing filters service")
	}
	if out.Net == nil {
		return EvmRpcAPI{}, errors.New("localnet RPC APIs missing net service")
	}
	if out.Debug == nil {
		return EvmRpcAPI{}, errors.New("localnet RPC APIs missing debug service")
	}

	return out, nil
}

func newClientCtx() (client.Context, error) {
	encCfg := app.MakeEncodingConfig()

	kb, err := keyring.New(
		sdk.KeyringServiceName(),
		keyring.BackendTest,
		app.DefaultNodeHome,
		os.Stdin,
		encCfg.Codec,
	)
	if err != nil {
		return client.Context{}, fmt.Errorf("create localnet keyring: %w", err)
	}

	rpcClient, err := client.NewClientFromNode(NodeURI)
	if err != nil {
		return client.Context{}, fmt.Errorf("connect localnet RPC client: %w", err)
	}

	return client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithKeyringDir(app.DefaultNodeHome).
		WithKeyring(kb).
		WithFromName(KeyName).
		WithFromAddress(nutil.LocalnetValAddr).
		WithChainID(ChainID).
		WithNodeURI(NodeURI).
		WithClient(rpcClient).
		WithBroadcastMode(flags.BroadcastSync).
		WithOutput(io.Discard), nil
}

func (c CLI) ExecQueryCmd(
	cmd *cobra.Command,
	args []string,
	result codec.ProtoMarshaler,
) error {
	renderedCmd := c.RenderQueryCmd(cmd, args)
	args = c.queryArgs(args)

	out, err := clitestutil.ExecTestCLICmd(c.ClientCtx, cmd, args)
	if err != nil {
		return fmt.Errorf("failed to execute query %s: %w: %s", renderedCmd, err, out.String())
	}
	return c.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), result)
}

func (c CLI) ExecTxCmd(
	cmd *cobra.Command,
	args []string,
	opts ...TxOption,
) (*sdk.TxResponse, error) {
	renderedCmd := c.RenderTxCmd(cmd, args, opts...)
	args = c.txArgs(args, opts...)

	out, err := clitestutil.ExecTestCLICmd(c.ClientCtx, cmd, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tx %s: %w: %s", renderedCmd, err, out.String())
	}

	txResp := new(sdk.TxResponse)
	if err := c.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp); err != nil {
		return nil, fmt.Errorf("failed to decode tx response for %s: %w: %s", renderedCmd, err, out.String())
	}
	if txResp.Code != types.CodeTypeOK {
		return nil, fmt.Errorf("tx failed for %s with code %d: %s", renderedCmd, txResp.Code, txResp.RawLog)
	}
	if txResp.TxHash == "" {
		return txResp, nil
	}

	deliveredResp, err := c.WaitForTx(txResp.TxHash)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for tx %s (%s): %w", txResp.TxHash, renderedCmd, err)
	}
	if deliveredResp.Code != types.CodeTypeOK {
		return nil, fmt.Errorf("delivered tx failed for %s with code %d: %s", renderedCmd, deliveredResp.Code, deliveredResp.RawLog)
	}

	return deliveredResp, nil
}

func (c CLI) RenderQueryCmd(cmd *cobra.Command, args []string) string {
	return renderNibidCmd("q", cmd, c.queryArgs(args))
}

func (c CLI) RenderTxCmd(
	cmd *cobra.Command,
	args []string,
	opts ...TxOption,
) string {
	return renderNibidCmd("tx", cmd, c.txArgs(args, opts...))
}

func (c CLI) queryArgs(args []string) []string {
	argsCopy := append([]string(nil), args...)
	return append(argsCopy,
		fmt.Sprintf("--%s=%s", cmtcli.OutputFlag, "json"),
		fmt.Sprintf("--%s=%s", flags.FlagNode, c.NodeURI),
	)
}

func (c CLI) txArgs(args []string, opts ...TxOption) []string {
	txOpts := c.defaultTxOptions()
	for _, opt := range opts {
		opt(&txOpts)
	}

	argsCopy := append([]string(nil), args...)
	txArgs := append(argsCopy,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, txOpts.fromName),
		fmt.Sprintf("--%s=%s", flags.FlagFees, txOpts.txFee),
		fmt.Sprintf("--%s=%s", flags.FlagGas, txOpts.txGas),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagChainID, ChainID),
		fmt.Sprintf("--%s=%s", flags.FlagKeyringBackend, keyring.BackendTest),
		fmt.Sprintf("--%s=%s", flags.FlagNode, c.NodeURI),
		fmt.Sprintf("--%s=%s", cmtcli.OutputFlag, "json"),
	)
	if txOpts.gasAdjustment != "" {
		txArgs = append(txArgs, fmt.Sprintf("--%s=%s", flags.FlagGasAdjustment, txOpts.gasAdjustment))
	}
	return txArgs
}

func (c CLI) defaultTxOptions() txOptions {
	return txOptions{
		fromName: c.FromName,
		txFee:    c.TxFee,
		txGas:    c.TxGas,
	}
}

func renderNibidCmd(verb string, cmd *cobra.Command, args []string) string {
	parts := []string{"nibid"}
	if verb != "" {
		parts = append(parts, verb)
	}
	if shouldRenderCmdName(verb, cmd) {
		parts = append(parts, cmd.Name())
	}
	for _, arg := range args {
		parts = append(parts, quoteShellArg(arg))
	}
	return strings.Join(parts, " ")
}

func shouldRenderCmdName(verb string, cmd *cobra.Command) bool {
	cmdName := cmd.Name()
	if cmdName == verb {
		return false
	}
	if verb == "q" && cmdName == "query" {
		return false
	}
	return true
}

func quoteShellArg(arg string) string {
	if arg == "" {
		return "''"
	}

	if !strings.ContainsAny(arg, " \t\n'\"\\$`;&|<>()[]{}*?!#~") {
		return arg
	}

	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

func (c CLI) WaitForTx(txHash string) (*sdk.TxResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		txResp, err := QueryTx(c.ClientCtx, txHash)
		if err == nil {
			return txResp, nil
		}
		lastErr = err
		if attempt < 2 {
			if waitErr := c.WaitForNextBlock(); waitErr != nil {
				return nil, fmt.Errorf("failed waiting for tx %s block inclusion: %w", txHash, waitErr)
			}
		}
	}
	return nil, fmt.Errorf("failed to query tx %s after waiting two blocks: %w", txHash, lastErr)
}

func (c CLI) LatestHeight() (int64, error) {
	if c.ClientCtx.Client == nil {
		return 0, errors.New("localnet client context has no RPC client")
	}

	status, err := c.ClientCtx.Client.Status(context.Background())
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

func (c CLI) WaitForHeight(h int64) (int64, error) {
	return c.WaitForHeightWithTimeout(h, 5*time.Minute)
}

func (c CLI) WaitForHeightWithTimeout(h int64, timeout time.Duration) (int64, error) {
	if c.ClientCtx.Client == nil {
		return 0, errors.New("localnet client context has no RPC client")
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var latestHeight int64
	for {
		select {
		case <-timer.C:
			return latestHeight, errors.New("timeout exceeded waiting for localnet block")
		case <-ticker.C:
			status, err := c.ClientCtx.Client.Status(context.Background())
			if err == nil && status != nil {
				latestHeight = status.SyncInfo.LatestBlockHeight
				if latestHeight >= h {
					return latestHeight, nil
				}
			}
		}
	}
}

func (c CLI) WaitForNextBlockVerbose() (int64, error) {
	lastBlock, err := c.LatestHeight()
	if err != nil {
		return -1, err
	}

	newBlock := lastBlock + 1
	_, err = c.WaitForHeight(newBlock)
	if err != nil {
		return lastBlock, err
	}

	return newBlock, nil
}

func (c CLI) WaitForNextBlock() error {
	_, err := c.WaitForNextBlockVerbose()
	return err
}
