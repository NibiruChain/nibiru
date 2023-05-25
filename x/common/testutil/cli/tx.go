package cli

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
)

type ExecTxOption func(*execTxOptions)

func WithTxFees(feeCoins sdk.Coins) ExecTxOption {
	return func(options *execTxOptions) {
		options.fees = feeCoins
	}
}

func WithTxSkipConfirmation(skipConfirmation bool) ExecTxOption {
	return func(options *execTxOptions) {
		options.skipConfirmation = skipConfirmation
	}
}

func WithTxBroadcastMode(broadcastMode string) ExecTxOption {
	return func(options *execTxOptions) {
		options.broadcastMode = broadcastMode
	}
}

// WithTxCanFail will not make ExecTx return an error
// in case the response code of the TX is not ok.
func WithTxCanFail(canFail bool) ExecTxOption {
	return func(options *execTxOptions) {
		options.canFail = canFail
	}
}

func WithKeyringBackend(keyringBackend string) ExecTxOption {
	return func(options *execTxOptions) {
		options.keyringBackend = keyringBackend
	}
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
		fees:             sdk.NewCoins(sdk.NewCoin(denoms.NIBI, sdk.NewInt(10))),
		gas:              2000000,
		skipConfirmation: true,
		broadcastMode:    flags.BroadcastBlock,
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

	rawResp, err := cli.ExecTestCLICmd(clientCtx, cmd, args)
	if err != nil {
		return nil, err
	}

	resp := new(sdk.TxResponse)
	err = clientCtx.Codec.UnmarshalJSON(rawResp.Bytes(), resp)
	if err != nil {
		return nil, err
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

	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, sdk.NewInt(1))))
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

	respRaw, err := rpc.BroadcastTxCommit(context.Background(), txBytes)
	if err != nil {
		return nil, err
	}

	if !respRaw.CheckTx.IsOK() {
		return nil, fmt.Errorf("tx failed: %s", respRaw.CheckTx.Log)
	}
	if !respRaw.DeliverTx.IsOK() {
		return nil, fmt.Errorf("tx failed: %s", respRaw.DeliverTx.Log)
	}

	return sdk.NewResponseFormatBroadcastTxCommit(respRaw), nil
}
