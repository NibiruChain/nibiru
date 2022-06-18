package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/x/common"
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

type execTxOptions struct {
	fees             sdk.Coins
	skipConfirmation bool
	broadcastMode    string
	canFail          bool
}

func ExecTx(network *Network, cmd *cobra.Command, txSender sdk.AccAddress, args []string, opt ...ExecTxOption) (*sdk.TxResponse, error) {
	if len(network.Validators) == 0 {
		return nil, fmt.Errorf("invalid network")
	}

	options := execTxOptions{
		fees:             sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))),
		skipConfirmation: true,
		broadcastMode:    flags.BroadcastBlock,
	}

	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagFrom, txSender))

	for _, o := range opt {
		o(&options)
	}

	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, options.broadcastMode))
	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagFees, options.fees))
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
