package cli

import (
	"context"
	sdkmath "cosmossdk.io/math"
	"fmt"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
)

type ExecTxOption func(*execTxOptions)

func WithTxOptions(newOptions TxOptionChanges) ExecTxOption {
	return func(options *execTxOptions) {
		if newOptions.BroadcastMode != nil {
			options.BroadcastMode = *newOptions.BroadcastMode
		}
		if newOptions.CanFail != nil {
			options.CanFail = *newOptions.CanFail
		}
		if newOptions.Fees != nil {
			options.Fees = *newOptions.Fees
		}
		if newOptions.Gas != nil {
			options.Gas = *newOptions.Gas
		}
		if newOptions.KeyringBackend != nil {
			options.KeyringBackend = *newOptions.KeyringBackend
		}
		if newOptions.SkipConfirmation != nil {
			options.SkipConfirmation = *newOptions.SkipConfirmation
		}
	}
}

type TxOptionChanges struct {
	BroadcastMode    *string
	CanFail          *bool
	Fees             *sdk.Coins
	Gas              *int64
	KeyringBackend   *string
	SkipConfirmation *bool
}

type execTxOptions struct {
	BroadcastMode    string
	CanFail          bool
	Fees             sdk.Coins
	Gas              int64
	KeyringBackend   string
	SkipConfirmation bool
}

var DEFAULT_TX_OPTIONS = execTxOptions{
	Fees:             sdk.NewCoins(sdk.NewCoin(denoms.NIBI, sdkmath.NewInt(1000))),
	Gas:              2000000,
	SkipConfirmation: true,
	BroadcastMode:    flags.BroadcastSync,
	CanFail:          false,
	KeyringBackend:   keyring.BackendTest,
}

func (network *Network) ExecTxCmd(
	cmd *cobra.Command, from sdk.AccAddress, args []string, opts ...ExecTxOption,
) (*sdk.TxResponse, error) {
	if len(network.Validators) == 0 {
		return nil, fmt.Errorf("invalid network")
	}

	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagFrom, from))

	options := DEFAULT_TX_OPTIONS

	for _, opt := range opts {
		opt(&options)
	}

	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, options.BroadcastMode))
	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagFees, options.Fees))
	args = append(args, fmt.Sprintf("--%s=%d", flags.FlagGas, options.Gas))
	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagKeyringBackend, options.KeyringBackend))
	switch options.SkipConfirmation {
	case true:
		args = append(args, fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation))
	case false:
		args = append(args, fmt.Sprintf("--%s=false", flags.FlagSkipConfirmation))
	}

	clientCtx := network.Validators[0].ClientCtx

	rawResp, err := sdktestutil.ExecTestCLICmd(clientCtx, cmd, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tx: %w", err)
	}

	err = network.WaitForNextBlock()
	if err != nil {
		return nil, err
	}

	txResp := new(sdk.TxResponse)
	clientCtx.Codec.MustUnmarshalJSON(rawResp.Bytes(), txResp)
	resp, err := QueryTx(clientCtx, txResp.TxHash)
	if err != nil {
		return nil, fmt.Errorf("failed to query tx: %w", err)
	}

	if options.CanFail {
		return resp, nil
	}

	if resp.Code != types.CodeTypeOK {
		return nil, fmt.Errorf("tx failed with code %d: %s", resp.Code, resp.RawLog)
	}

	return resp, nil
}

func (chain *Network) BroadcastMsgs(
	from sdk.AccAddress, msgs ...sdk.Msg,
) (*sdk.TxResponse, error) {
	cfg := chain.Config
	kb, info, err := chain.keyBaseAndInfoForAddr(from)
	if err != nil {
		return nil, err
	}

	rpc := chain.Validators[0].RPCClient
	txBuilder := cfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}

	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, sdkmath.NewInt(1000))))
	txBuilder.SetGasLimit(uint64(1 * common.TO_MICRO))

	acc, err := cfg.AccountRetriever.GetAccount(chain.Validators[0].ClientCtx, from)
	if err != nil {
		return nil, err
	}

	txFactory := tx.Factory{}
	txFactory = txFactory.
		WithChainID(cfg.ChainID).
		WithKeybase(kb).
		WithTxConfig(cfg.TxConfig).
		WithAccountRetriever(cfg.AccountRetriever).
		WithAccountNumber(acc.GetAccountNumber()).
		WithSequence(acc.GetSequence())

	err = tx.Sign(context.Background(), txFactory, info.Name, txBuilder, true)
	if err != nil {
		return nil, err
	}

	txBytes, err := cfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	respRaw, err := rpc.BroadcastTxSync(context.Background(), txBytes)
	if err != nil {
		return nil, err
	}

	return sdk.NewResponseFormatBroadcastTx(respRaw), err
}
