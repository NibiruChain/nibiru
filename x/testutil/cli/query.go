package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/NibiruChain/nibiru/x/perp/client/cli"
	"github.com/NibiruChain/nibiru/x/perp/types"

	"github.com/NibiruChain/nibiru/x/common"
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

func QueryBaseAssetPrice(ctx client.Context, pair common.AssetPair, direction string, amount string) (vpooltypes.QueryBaseAssetResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		vpoolcli.CmdGetBaseAssetPrice(),
		[]string{pair.String(), direction, amount, fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
	)
	if err != nil {
		return vpooltypes.QueryBaseAssetResponse{}, err
	}

	var queryResp vpooltypes.QueryBaseAssetResponse
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
