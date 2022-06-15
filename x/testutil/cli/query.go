package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/NibiruChain/nibiru/x/perp/client/cli"
	"github.com/NibiruChain/nibiru/x/perp/types"
	pricefeedcli "github.com/NibiruChain/nibiru/x/pricefeed/client/cli"

	"github.com/NibiruChain/nibiru/x/common"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	vpoolcli "github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func QueryVpoolReserveAssets(ctx client.Context, pair common.AssetPair) (vpooltypes.QueryReserveAssetsResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(ctx, vpoolcli.CmdGetVpoolReserveAssets(), []string{pair.String(), fmt.Sprintf("--%s=json", tmcli.OutputFlag)})
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

func QueryTraderPosition(ctx client.Context, pair common.AssetPair, trader sdk.AccAddress) (types.QueryTraderPositionResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		cli.CmdQueryPosition(),
		[]string{trader.String(), pair.String(), fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
	)
	if err != nil {
		return types.QueryTraderPositionResponse{}, err
	}

	var queryResp types.QueryTraderPositionResponse
	err = ctx.Codec.UnmarshalJSON(out.Bytes(), &queryResp)
	if err != nil {
		return types.QueryTraderPositionResponse{}, err
	}

	return queryResp, nil
}

func QueryPrice(ctx client.Context, pairID string) (pricefeedtypes.QueryPriceResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		pricefeedcli.CmdPrice(),
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
		pricefeedcli.CmdRawPrices(),
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
