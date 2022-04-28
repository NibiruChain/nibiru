package testutil

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	dexcli "github.com/NibiruChain/nibiru/x/dex/client/cli"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// commonArgs is args for CLI test commands.
var commonArgs = []string{
	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))).String()),
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
	extraArgs ...string,
) (testutil.BufferWriter, error) {
	args := []string{}

	jsonFile := testutil.WriteToNewTempFile(t,
		fmt.Sprintf(`
		{
		  "weights": "%s",
		  "initial-deposit": "%s",
		  "swap-fee": "%s",
		  "exit-fee": "%s"
		}
		`,
			tokenWeights,
			initialDeposit,
			swapFee,
			exitFee,
		),
	)

	args = append(args,
		fmt.Sprintf("--%s=%s", dexcli.FlagPoolFile, jsonFile.Name()),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, owner.String()),
		fmt.Sprintf("--%s=%d", flags.FlagGas, 300000),
	)

	args = append(args, commonArgs...)
	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, dexcli.CmdCreatePool(), args)
}

// ExecMsgJoinPool broadcast a join pool message.
func ExecMsgJoinPool(
	t *testing.T,
	clientCtx client.Context,
	poolId uint64,
	sender fmt.Stringer,
	tokensIn string,
	extraArgs ...string,
) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, poolId),
		fmt.Sprintf("--%s=%s", dexcli.FlagTokensIn, tokensIn),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, sender.String()),
		fmt.Sprintf("--%s=%d", flags.FlagGas, 300000),
	}

	args = append(args, commonArgs...)
	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, dexcli.CmdJoinPool(), args)
}

// ExecMsgExitPool broadcast an exit pool message.
func ExecMsgExitPool(
	t *testing.T,
	clientCtx client.Context,
	poolId uint64,
	sender fmt.Stringer,
	poolSharesOut string,
	extraArgs ...string,
) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, poolId),
		fmt.Sprintf("--%s=%s", dexcli.FlagPoolSharesOut, poolSharesOut),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, sender.String()),
		fmt.Sprintf("--%s=%d", flags.FlagGas, 300000),
	}

	args = append(args, commonArgs...)
	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, dexcli.CmdExitPool(), args)
}

// ExecMsgSwapAssets broadcast a swap assets message.
func ExecMsgSwapAssets(
	t *testing.T,
	clientCtx client.Context,
	poolId uint64,
	sender fmt.Stringer,
	tokenIn string,
	tokenOutDenom string,
	extraArgs ...string,
) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--%s=%d", dexcli.FlagPoolId, poolId),
		fmt.Sprintf("--%s=%s", dexcli.FlagTokenIn, tokenIn),
		fmt.Sprintf("--%s=%s", dexcli.FlagTokenOutDenom, tokenOutDenom),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, sender.String()),
		fmt.Sprintf("--%s=%d", flags.FlagGas, 300_000),
	}

	args = append(args, commonArgs...)
	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, dexcli.CmdSwapAssets(), args)
}
