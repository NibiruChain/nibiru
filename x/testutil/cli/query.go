package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/NibiruChain/nibiru/x/common"
	perpcli "github.com/NibiruChain/nibiru/x/perp/client/cli"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	pricefeedcli "github.com/NibiruChain/nibiru/x/pricefeed/client/cli"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	vpoolcli "github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
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
func ExecQuery(network *Network, cmd *cobra.Command, args []string, result codec.ProtoMarshaler, opts ...ExecQueryOption) error {
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
	if len(network.Validators) == 0 {
		return fmt.Errorf("invalid network")
	}

	clientCtx := network.Validators[0].ClientCtx

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

func QueryVpoolReserveAssets(ctx client.Context, pair common.AssetPair,
) (vpooltypes.QueryReserveAssetsResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		vpoolcli.CmdGetVpoolReserveAssets(),
		[]string{pair.String(),
			fmt.Sprintf("--%s=json", tmcli.OutputFlag)})
	if err != nil {
		return vpooltypes.QueryReserveAssetsResponse{}, err
	}

	var queryResp vpooltypes.QueryReserveAssetsResponse
	err = ctx.Codec.UnmarshalJSON(out.Bytes(), &queryResp)
	if err != nil {
		return vpooltypes.QueryReserveAssetsResponse{}, err
	}

	return queryResp, nil
}

func QueryBaseAssetPrice(ctx client.Context, pair common.AssetPair, direction string, amount string) (vpooltypes.QueryBaseAssetPriceResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		vpoolcli.CmdGetBaseAssetPrice(),
		[]string{pair.String(), direction, amount, fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
	)
	if err != nil {
		return vpooltypes.QueryBaseAssetPriceResponse{}, err
	}

	var queryResp vpooltypes.QueryBaseAssetPriceResponse
	ctx.Codec.MustUnmarshalJSON(out.Bytes(), &queryResp)

	return queryResp, nil
}

func QueryTraderPosition(ctx client.Context, pair common.AssetPair, trader sdk.AccAddress) (perptypes.QueryTraderPositionResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		perpcli.CmdQueryPosition(),
		[]string{trader.String(), pair.String(), fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
	)
	if err != nil {
		return perptypes.QueryTraderPositionResponse{}, err
	}

	var queryResp perptypes.QueryTraderPositionResponse
	err = ctx.Codec.UnmarshalJSON(out.Bytes(), &queryResp)
	if err != nil {
		return perptypes.QueryTraderPositionResponse{}, err
	}

	return queryResp, nil
}

func QueryFundingRates(ctx client.Context, pair common.AssetPair) (perptypes.QueryFundingRatesResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		perpcli.CmdQueryFundingRates(),
		[]string{pair.String(), fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
	)
	if err != nil {
		return perptypes.QueryFundingRatesResponse{}, err
	}

	var queryResp perptypes.QueryFundingRatesResponse
	err = ctx.Codec.UnmarshalJSON(out.Bytes(), &queryResp)
	if err != nil {
		return perptypes.QueryFundingRatesResponse{}, err
	}

	return queryResp, nil
}

func QueryPrice(ctx client.Context, pairID string) (pricefeedtypes.QueryPriceResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		pricefeedcli.CmdQueryPrice(),
		[]string{pairID, fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
	)
	if err != nil {
		return pricefeedtypes.QueryPriceResponse{}, err
	}

	var queryResp pricefeedtypes.QueryPriceResponse
	err = ctx.Codec.UnmarshalJSON(out.Bytes(), &queryResp)
	if err != nil {
		return pricefeedtypes.QueryPriceResponse{}, err
	}

	return queryResp, nil
}

func QueryRawPrice(ctx client.Context, pairID string) (pricefeedtypes.QueryRawPricesResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		pricefeedcli.CmdQueryRawPrices(),
		[]string{pairID, fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
	)
	if err != nil {
		return pricefeedtypes.QueryRawPricesResponse{}, err
	}

	var queryResp pricefeedtypes.QueryRawPricesResponse
	err = ctx.Codec.UnmarshalJSON(out.Bytes(), &queryResp)
	if err != nil {
		return pricefeedtypes.QueryRawPricesResponse{}, err
	}

	return queryResp, nil
}
