package testnetwork

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
)

const (
	LocalnetChainID = "nibiru-localnet-0"
	LocalnetKeyName = "validator"
	LocalnetNodeURI = "http://localhost:26657"
	LocalnetTxFee   = "1000" + appconst.DENOM_UNIBI
	LocalnetTxGas   = "5000000"
)

type LocalnetCLI struct {
	ClientCtx client.Context
	FromName  string
	FromAddr  sdk.AccAddress
	NodeURI   string
	TxFee     string
	TxGas     string
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
	args = append(args,
		fmt.Sprintf("--%s=%s", cli.OutputFlag, "json"),
		fmt.Sprintf("--%s=%s", flags.FlagNode, c.NodeURI),
	)

	out, err := clitestutil.ExecTestCLICmd(c.ClientCtx, cmd, args)
	if err != nil {
		return err
	}
	return c.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), result)
}

func (c LocalnetCLI) ExecTxCmd(
	cmd *cobra.Command,
	args []string,
) (*sdk.TxResponse, error) {
	args = append(args,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, c.FromName),
		fmt.Sprintf("--%s=%s", flags.FlagFees, c.TxFee),
		fmt.Sprintf("--%s=%s", flags.FlagGas, c.TxGas),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagChainID, LocalnetChainID),
		fmt.Sprintf("--%s=%s", flags.FlagKeyringBackend, keyring.BackendTest),
		fmt.Sprintf("--%s=%s", flags.FlagNode, c.NodeURI),
		fmt.Sprintf("--%s=%s", cli.OutputFlag, "json"),
	)

	out, err := clitestutil.ExecTestCLICmd(c.ClientCtx, cmd, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tx: %w: %s", err, out.String())
	}

	txResp := new(sdk.TxResponse)
	if err := c.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), txResp); err != nil {
		return nil, fmt.Errorf("failed to decode tx response: %w: %s", err, out.String())
	}
	if txResp.Code != types.CodeTypeOK {
		return nil, fmt.Errorf("tx failed with code %d: %s", txResp.Code, txResp.RawLog)
	}
	if txResp.TxHash == "" {
		return txResp, nil
	}

	deliveredResp, err := c.WaitForTx(txResp.TxHash)
	if err != nil {
		return nil, err
	}
	if deliveredResp.Code != types.CodeTypeOK {
		return nil, fmt.Errorf("tx failed with code %d: %s", deliveredResp.Code, deliveredResp.RawLog)
	}

	return deliveredResp, nil
}

func (c LocalnetCLI) WaitForTx(txHash string) (*sdk.TxResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		txResp, err := QueryTx(c.ClientCtx, txHash)
		if err == nil {
			return txResp, nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("failed to query tx %s: %w", txHash, lastErr)
}
