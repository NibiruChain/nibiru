package testutil

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/spot/client/cli"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

// commonArgs is args for CLI test commands.
var commonArgs = []string{
	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(denoms.NIBI, sdk.NewInt(10))).String()),
}

// ExecMsgCreatePool broadcast a pool creation message.
func ExecMsgCreatePool(
	t *testing.T,
	clientCtx client.Context,
	owner fmt.Stringer,
	tokenWeights string,
	initialDeposit string,
	swapFee string,
	exitFee string,
	poolType string,
	amplification string,
	extraArgs ...string,
) (testutil.BufferWriter, error) {
	args := []string{}

	jsonFile := testutil.WriteToNewTempFile(t,
		fmt.Sprintf(`
		{
		  "weights": "%s",
		  "initial-deposit": "%s",
		  "swap-fee": "%s",
		  "exit-fee": "%s",
		  "pool-type": "%s",
		  "amplification": "%s"
		}
		`,
			tokenWeights,
			initialDeposit,
			swapFee,
			exitFee,
			poolType,
			amplification,
		),
	)

	args = append(args,
		fmt.Sprintf("--%s=%s", cli.FlagPoolFile, jsonFile.Name()),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, owner.String()),
		fmt.Sprintf("--%s=%d", flags.FlagGas, 300000),
	)

	args = append(args, commonArgs...)
	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, cli.CmdCreatePool(), args)
}

// ExecMsgJoinPool broadcast a join pool message.
func ExecMsgJoinPool(
	clientCtx client.Context,
	poolId uint64,
	sender fmt.Stringer,
	tokensIn string,
	useAllCoins string,
	extraArgs ...string,
) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--%s=%d", cli.FlagPoolId, poolId),
		fmt.Sprintf("--%s=%s", cli.FlagTokensIn, tokensIn),
		fmt.Sprintf("--%s=%s", cli.FlagUseAllCoins, useAllCoins),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, sender.String()),
		fmt.Sprintf("--%s=%d", flags.FlagGas, 300000),
	}

	args = append(args, commonArgs...)
	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, cli.CmdJoinPool(), args)
}

// ExecMsgExitPool broadcast an exit pool message.
func ExecMsgExitPool(
	clientCtx client.Context,
	poolId uint64,
	sender fmt.Stringer,
	poolSharesOut string,
	extraArgs ...string,
) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--%s=%d", cli.FlagPoolId, poolId),
		fmt.Sprintf("--%s=%s", cli.FlagPoolSharesOut, poolSharesOut),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, sender.String()),
		fmt.Sprintf("--%s=%d", flags.FlagGas, 300000),
	}

	args = append(args, commonArgs...)
	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, cli.CmdExitPool(), args)
}

// ExecMsgSwapAssets broadcast a swap assets message.
func ExecMsgSwapAssets(
	clientCtx client.Context,
	poolId uint64,
	sender fmt.Stringer,
	tokenIn string,
	tokenOutDenom string,
	extraArgs ...string,
) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--%s=%d", cli.FlagPoolId, poolId),
		fmt.Sprintf("--%s=%s", cli.FlagTokenIn, tokenIn),
		fmt.Sprintf("--%s=%s", cli.FlagTokenOutDenom, tokenOutDenom),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, sender.String()),
		fmt.Sprintf("--%s=%d", flags.FlagGas, 300_000),
	}

	args = append(args, commonArgs...)
	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, cli.CmdSwapAssets(), args)
}

// WhitelistGenesisAssets given a testapp.GenesisState includes the whitelisted assets into spot Whitelisted assets.
func WhitelistGenesisAssets(state app.GenesisState, assets []string) app.GenesisState {
	encConfig := app.MakeTestEncodingConfig()

	jsonState := state[types.ModuleName]

	var genesis types.GenesisState
	encConfig.Marshaler.MustUnmarshalJSON(jsonState, &genesis)
	genesis.Params.WhitelistedAsset = assets

	json, _ := encConfig.Marshaler.MarshalJSON(&genesis)
	state[types.ModuleName] = json

	return state
}

// ExtractPoolIDFromCreatePoolResponse extracts the created PoolID from a MsgCreatePool command.
func ExtractPoolIDFromCreatePoolResponse(codec codec.Codec, out testutil.BufferWriter) (uint64, error) {
	resp := &sdk.TxResponse{}
	err := codec.UnmarshalJSON(out.Bytes(), resp)
	if err != nil {
		return 0, err
	}

	decodedResult, err := hex.DecodeString(resp.Data)
	if err != nil {
		return 0, err
	}

	respData := sdk.TxMsgData{}
	err = codec.Unmarshal(decodedResult, &respData)
	if err != nil {
		return 0, err
	}

	if len(respData.Data) < 1 {
		return 0, fmt.Errorf("invalid response")
	}

	var createPoolResponse types.MsgCreatePoolResponse
	err = codec.Unmarshal(respData.Data[0].Data, &createPoolResponse)
	if err != nil {
		return 0, err
	}

	return createPoolResponse.PoolId, nil
}
