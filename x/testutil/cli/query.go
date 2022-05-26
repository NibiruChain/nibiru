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
	cli2 "github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func QueryVpoolReserveAssets(ctx client.Context, pair common.TokenPair) (vpooltypes.QueryReserveAssetsResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(ctx, cli2.CmdGetVpoolReserveAssets(), []string{string(pair), fmt.Sprintf("--%s=json", tmcli.OutputFlag)})
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

func QueryTraderPosition(ctx client.Context, pair common.TokenPair, trader sdk.AccAddress) (types.QueryTraderPositionResponse, error) {
	out, err := clitestutil.ExecTestCLICmd(
		ctx,
		cli.CmdQueryPosition(),
		[]string{trader.String(), string(pair), fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
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
