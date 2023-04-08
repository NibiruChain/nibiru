package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/NibiruChain/nibiru/x/common/asset"
	oraclecli "github.com/NibiruChain/nibiru/x/oracle/client/cli"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	perpammcli "github.com/NibiruChain/nibiru/x/perp/amm/cli"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	perpcli "github.com/NibiruChain/nibiru/x/perp/client/cli"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
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

// ExecQuery executes a CLI query onto the provided Network.
func ExecQuery(clientCtx client.Context, cmd *cobra.Command, args []string, result codec.ProtoMarshaler, opts ...ExecQueryOption) error {
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

func QueryMarketReserveAssets(clientCtx client.Context, pair asset.Pair,
) (*perpammtypes.QueryReserveAssetsResponse, error) {
	var queryResp perpammtypes.QueryReserveAssetsResponse
	if err := ExecQuery(clientCtx, perpammcli.CmdGetMarketReserveAssets(), []string{pair.String()}, &queryResp); err != nil {
		return nil, err
	}
	return &queryResp, nil
}

func QueryOracleExchangeRate(clientCtx client.Context, pair asset.Pair) (*oracletypes.QueryExchangeRateResponse, error) {
	var queryResp oracletypes.QueryExchangeRateResponse
	if err := ExecQuery(clientCtx, oraclecli.GetCmdQueryExchangeRates(), []string{pair.String()}, &queryResp); err != nil {
		return nil, err
	}
	return &queryResp, nil
}

func QueryBaseAssetPrice(clientCtx client.Context, pair asset.Pair, direction string, amount string) (*perpammtypes.QueryBaseAssetPriceResponse, error) {
	var queryResp perpammtypes.QueryBaseAssetPriceResponse
	if err := ExecQuery(clientCtx, perpammcli.CmdGetBaseAssetPrice(), []string{pair.String(), direction, amount}, &queryResp); err != nil {
		return nil, err
	}
	return &queryResp, nil
}

func QueryPosition(ctx client.Context, pair asset.Pair, trader sdk.AccAddress) (*perptypes.QueryPositionResponse, error) {
	var queryResp perptypes.QueryPositionResponse
	if err := ExecQuery(ctx, perpcli.CmdQueryPosition(), []string{trader.String(), pair.String()}, &queryResp); err != nil {
		return nil, err
	}
	return &queryResp, nil
}

func QueryCumulativePremiumFraction(clientCtx client.Context, pair asset.Pair) (*perptypes.QueryCumulativePremiumFractionResponse, error) {
	var queryResp perptypes.QueryCumulativePremiumFractionResponse
	if err := ExecQuery(clientCtx, perpcli.CmdQueryCumulativePremiumFraction(), []string{pair.String()}, &queryResp); err != nil {
		return nil, err
	}
	return &queryResp, nil
}
