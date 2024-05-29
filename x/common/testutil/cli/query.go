package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/x/auth/client/cli"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	// oraclecli "github.com/NibiruChain/nibiru/x/oracle/client/cli"
	// oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	// sudocli "github.com/NibiruChain/nibiru/x/sudo/cli"
	// sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
)

// ExecQueryOption defines a type which customizes a CLI query operation.
type ExecQueryOption func(queryOption *queryOptions)

// queryOptions is an internal type which defines options.
type queryOptions struct {
	outputEncoding EncodingType
}

// EncodingType defines the encoding methodology for requests and responses.
type EncodingType int

const (
	// EncodingTypeJSON defines the types are JSON encoded or need to be encoded using JSON.
	EncodingTypeJSON = iota
	// EncodingTypeProto defines the types are proto encoded or need to be encoded using proto.
	EncodingTypeProto
)

// WithQueryEncodingType defines how the response of the CLI query should be decoded.
func WithQueryEncodingType(e EncodingType) ExecQueryOption {
	return func(queryOption *queryOptions) {
		queryOption.outputEncoding = e
	}
}

func (chain Network) ExecQuery(
	cmd *cobra.Command,
	args []string,
	result codec.ProtoMarshaler,
	opts ...ExecQueryOption,
) error {
	return ExecQuery(chain.Validators[0].ClientCtx, cmd, args, result, opts...)
}

// ExecQuery executes a CLI query onto the provided Network.
func ExecQuery(
	clientCtx client.Context,
	cmd *cobra.Command,
	args []string,
	result codec.ProtoMarshaler,
	opts ...ExecQueryOption,
) error {
	var options queryOptions
	for _, o := range opts {
		o(&options)
	}
	switch options.outputEncoding {
	case EncodingTypeJSON:
		args = append(args, fmt.Sprintf("--%s=json", tmcli.OutputFlag))
	case EncodingTypeProto:
		return fmt.Errorf("query proto encoding is not supported")
	default:
		return fmt.Errorf("unknown query encoding type %d", options.outputEncoding)
	}

	resultRaw, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
	if err != nil {
		return err
	}

	switch options.outputEncoding {
	case EncodingTypeJSON:
		return clientCtx.Codec.UnmarshalJSON(resultRaw.Bytes(), result)
	case EncodingTypeProto:
		return clientCtx.Codec.Unmarshal(resultRaw.Bytes(), result)
	default:
		return fmt.Errorf("unrecognized encoding option %v", options.outputEncoding)
	}
}

// func QueryOracleExchangeRate(clientCtx client.Context, pair asset.Pair) (*oracletypes.QueryExchangeRateResponse, error) {
// 	var queryResp oracletypes.QueryExchangeRateResponse
// 	if err := ExecQuery(clientCtx, oraclecli.GetCmdQueryExchangeRates(), []string{pair.String()}, &queryResp); err != nil {
// 		return nil, err
// 	}
// 	return &queryResp, nil
// }

func QueryTx(ctx client.Context, txHash string) (*sdk.TxResponse, error) {
	var queryResp sdk.TxResponse
	if err := ExecQuery(
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

// func QuerySudoers(clientCtx client.Context) (*sudotypes.QuerySudoersResponse, error) {
// 	var queryResp sudotypes.QuerySudoersResponse
// 	if err := ExecQuery(
// 		clientCtx,
// 		sudocli.CmdQuerySudoers(),
// 		[]string{},
// 		&queryResp,
// 	); err != nil {
// 		return nil, err
// 	}
// 	return &queryResp, nil
// }
