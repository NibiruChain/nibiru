package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/NibiruChain/nibiru/x/common"
	cli2 "github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func CliQueryVpoolReserveAssets(ctx client.Context, pair common.TokenPair) (vpooltypes.QueryReserveAssetsResponse, error) {
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
