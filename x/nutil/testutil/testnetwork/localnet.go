package testnetwork

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/cli"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
)

const (
	LocalnetChainID = "nibiru-localnet-0"
	LocalnetKeyName = "validator"
	LocalnetNodeURI = "http://localhost:26657"
	LocalnetEVMURI  = "http://127.0.0.1:8545"
	LocalnetTxFee   = "1000" + appconst.DENOM_UNIBI
	LocalnetTxGas   = "5000000"
)

type LocalnetBackend struct {
	ClientCtx     client.Context
	EthRpcBackend *rpcapi.Backend
	EvmRpcClient  *ethclient.Client
}

type LocalnetCLI struct {
	ClientCtx client.Context
	FromName  string
	FromAddr  sdk.AccAddress
	NodeURI   string
	TxFee     string
	TxGas     string
}

type LocalnetTxOption func(*localnetTxOptions)

type localnetTxOptions struct {
	fromName      string
	txFee         string
	txGas         string
	gasAdjustment string
}

func WithLocalnetTxFees(fees string) LocalnetTxOption {
	return func(opts *localnetTxOptions) {
		opts.txFee = fees
	}
}

func WithLocalnetTxGas(gas string) LocalnetTxOption {
	return func(opts *localnetTxOptions) {
		opts.txGas = gas
	}
}

func WithLocalnetTxGasAdjustment(gasAdjustment string) LocalnetTxOption {
	return func(opts *localnetTxOptions) {
		opts.gasAdjustment = gasAdjustment
	}
}

func NewLocalnetCLI() (LocalnetCLI, error) {
	clientCtx, err := NewLocalnetClientCtx()
	if err != nil {
		return LocalnetCLI{}, err
	}
	return LocalnetCLI{
		ClientCtx: clientCtx,
		FromName:  LocalnetKeyName,
		FromAddr:  nutil.LocalnetValAddr,
		NodeURI:   LocalnetNodeURI,
		TxFee:     LocalnetTxFee,
		TxGas:     LocalnetTxGas,
	}, nil
}

func NewLocalnetBackend() (LocalnetBackend, error) {
	clientCtx, err := NewLocalnetClientCtx()
	if err != nil {
		return LocalnetBackend{}, err
	}

	evmRpcClient, err := ethclient.Dial(LocalnetEVMURI)
	if err != nil {
		return LocalnetBackend{}, fmt.Errorf("connect localnet EVM RPC client: %w", err)
	}

	serverCtx := server.NewDefaultContext()
	serverCtx.Logger = cmtlog.NewNopLogger()

	return LocalnetBackend{
		ClientCtx:     clientCtx,
		EthRpcBackend: rpcapi.NewBackend(serverCtx, serverCtx.Logger, clientCtx, false, nil),
		EvmRpcClient:  evmRpcClient,
	}, nil
}

func NewLocalnetClientCtx() (client.Context, error) {
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

	rpcClient, err := client.NewClientFromNode(LocalnetNodeURI)
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
		WithFromName(LocalnetKeyName).
		WithFromAddress(nutil.LocalnetValAddr).
		WithChainID(LocalnetChainID).
		WithNodeURI(LocalnetNodeURI).
		WithClient(rpcClient).
		WithBroadcastMode(flags.BroadcastSync).
		WithOutput(io.Discard), nil
}

func (c LocalnetCLI) ExecQueryCmd(
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

func (c LocalnetCLI) ExecTxCmd(
	cmd *cobra.Command,
	args []string,
	opts ...LocalnetTxOption,
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

func (c LocalnetCLI) RenderQueryCmd(cmd *cobra.Command, args []string) string {
	return renderNibidCmd("q", cmd, c.queryArgs(args))
}

func (c LocalnetCLI) RenderTxCmd(
	cmd *cobra.Command,
	args []string,
	opts ...LocalnetTxOption,
) string {
	return renderNibidCmd("tx", cmd, c.txArgs(args, opts...))
}

func (c LocalnetCLI) queryArgs(args []string) []string {
	argsCopy := append([]string(nil), args...)
	return append(argsCopy,
		fmt.Sprintf("--%s=%s", cli.OutputFlag, "json"),
		fmt.Sprintf("--%s=%s", flags.FlagNode, c.NodeURI),
	)
}

func (c LocalnetCLI) txArgs(args []string, opts ...LocalnetTxOption) []string {
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
		fmt.Sprintf("--%s=%s", flags.FlagChainID, LocalnetChainID),
		fmt.Sprintf("--%s=%s", flags.FlagKeyringBackend, keyring.BackendTest),
		fmt.Sprintf("--%s=%s", flags.FlagNode, c.NodeURI),
		fmt.Sprintf("--%s=%s", cli.OutputFlag, "json"),
	)
	if txOpts.gasAdjustment != "" {
		txArgs = append(txArgs, fmt.Sprintf("--%s=%s", flags.FlagGasAdjustment, txOpts.gasAdjustment))
	}
	return txArgs
}

func (c LocalnetCLI) defaultTxOptions() localnetTxOptions {
	return localnetTxOptions{
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

func (c LocalnetCLI) WaitForTx(txHash string) (*sdk.TxResponse, error) {
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

func (c LocalnetCLI) LatestHeight() (int64, error) {
	if c.ClientCtx.Client == nil {
		return 0, errors.New("localnet client context has no RPC client")
	}

	status, err := c.ClientCtx.Client.Status(context.Background())
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

func (c LocalnetCLI) WaitForHeight(h int64) (int64, error) {
	return c.WaitForHeightWithTimeout(h, 5*time.Minute)
}

func (c LocalnetCLI) WaitForHeightWithTimeout(h int64, timeout time.Duration) (int64, error) {
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

func (c LocalnetCLI) WaitForNextBlockVerbose() (int64, error) {
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

func (c LocalnetCLI) WaitForNextBlock() error {
	_, err := c.WaitForNextBlockVerbose()
	return err
}
