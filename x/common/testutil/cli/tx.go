package cli

import (
	"context"
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
			options.broadcastMode = *newOptions.BroadcastMode
		}
		if newOptions.CanFail != nil {
			options.canFail = *newOptions.CanFail
		}
		if newOptions.Fees != nil {
			options.fees = *newOptions.Fees
		}
		if newOptions.Gas != nil {
			options.gas = *newOptions.Gas
		}
		if newOptions.KeyringBackend != nil {
			options.keyringBackend = *newOptions.KeyringBackend
		}
		if newOptions.SkipConfirmation != nil {
			options.skipConfirmation = *newOptions.SkipConfirmation
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
	fees             sdk.Coins
	gas              int64
	skipConfirmation bool
	broadcastMode    string
	canFail          bool
	keyringBackend   string
}

func ExecTx(network *Network, cmd *cobra.Command, txSender sdk.AccAddress, args []string, opt ...ExecTxOption) (*sdk.TxResponse, error) {
	if len(network.Validators) == 0 {
		return nil, fmt.Errorf("invalid network")
	}

	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagFrom, txSender))

	options := execTxOptions{
		fees:             sdk.NewCoins(sdk.NewCoin(denoms.NIBI, sdk.NewInt(1000))),
		gas:              2000000,
		skipConfirmation: true,
		broadcastMode:    flags.BroadcastSync,
		canFail:          false,
		keyringBackend:   keyring.BackendTest,
	}

	for _, o := range opt {
		o(&options)
	}

	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, options.broadcastMode))
	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagFees, options.fees))
	args = append(args, fmt.Sprintf("--%s=%d", flags.FlagGas, options.gas))
	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagKeyringBackend, options.keyringBackend))
	switch options.skipConfirmation {
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

	if options.canFail {
		return resp, nil
	}

	if resp.Code != types.CodeTypeOK {
		return nil, fmt.Errorf("tx failed with code %d: %s", resp.Code, resp.RawLog)
	}

	return resp, nil
}

func (n *Network) SendTx(addr sdk.AccAddress, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	cfg := n.Config
	kb, info, err := n.keyBaseAndInfoForAddr(addr)
	if err != nil {
		return nil, err
	}

	rpc := n.Validators[0].RPCClient
	txBuilder := cfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}

	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, sdk.NewInt(1000))))
	txBuilder.SetGasLimit(uint64(1 * common.TO_MICRO))

	acc, err := cfg.AccountRetriever.GetAccount(n.Validators[0].ClientCtx, addr)
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

	err = tx.Sign(txFactory, info.Name, txBuilder, true)
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

	return sdk.NewResponseFormatBroadcastTx(respRaw), nil
}
