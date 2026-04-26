package localnet

import (
	"fmt"

	cmtcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/spf13/cobra"
)

// ----------------------------------------------------------------------
// Query using CLI commands
// ----------------------------------------------------------------------

// ExecQueryCmd executes a CLI query onto the provided Network.
func ExecQueryCmd(
	clientCtx client.Context,
	cmd *cobra.Command,
	args []string,
	result codec.ProtoMarshaler,
) error {
	args = append(args, fmt.Sprintf("--%s=json", cmtcli.OutputFlag))

	resultRaw, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
	if err != nil {
		return err
	}

	return clientCtx.Codec.UnmarshalJSON(resultRaw.Bytes(), result)
}

func QueryTx(ctx client.Context, txHash string) (*sdk.TxResponse, error) {
	var queryResp sdk.TxResponse
	if err := ExecQueryCmd(
		ctx,
		cli.QueryTxCmd(),
		[]string{
			txHash,
		},
		&queryResp,
	); err != nil {
		return nil, err
	}

	return &queryResp, nil
}
