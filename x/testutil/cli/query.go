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
	cli2 "github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func QueryVpoolReserveAssets(ctx client.Context, pair common.AssetPair) (vpooltypes.QueryReserveAssetsResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(ctx, cli2.CmdGetVpoolReserveAssets(), []string{pair.String(), fmt.Sprintf("--%s=json", tmcli.OutputFlag)})
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

func QueryPrice(ctx client.Context, token0 string, token1 string) (pricefeedtypes.QueryPriceResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		pricefeedcli.CmdRawPrices(),
		[]string{token0 + ":" + token1, fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
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
