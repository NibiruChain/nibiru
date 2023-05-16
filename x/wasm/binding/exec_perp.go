package binding

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/v1/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/v1/types"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
)

type ExecutorPerp struct {
	Perp perpkeeper.Keeper
}

func (exec *ExecutorPerp) MsgServer() perptypes.MsgServer {
	return perpkeeper.NewMsgServerImpl(exec.Perp)
}

func (exec *ExecutorPerp) OpenPosition(
	cwMsg *cw_struct.OpenPosition, ctx sdk.Context,
) (
	sdkResp *perptypes.MsgOpenPositionResponse, err error,
) {
	if cwMsg == nil {
		return sdkResp, wasmvmtypes.InvalidRequest{Err: "null open position msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return sdkResp, err
	}

	var side perpammtypes.Direction
	if cwMsg.IsLong {
		side = perpammtypes.Direction_LONG
	} else {
		side = perpammtypes.Direction_SHORT
	}

	sdkMsg := &perptypes.MsgOpenPosition{
		Sender:               cwMsg.Sender,
		Pair:                 pair,
		Side:                 side,
		QuoteAssetAmount:     cwMsg.QuoteAmount,
		Leverage:             cwMsg.Leverage,
		BaseAssetAmountLimit: cwMsg.BaseAmountLimit,
	}

	goCtx := sdk.WrapSDKContext(ctx)
	return exec.MsgServer().OpenPosition(goCtx, sdkMsg)
}

func (exec *ExecutorPerp) ClosePosition(
	cwMsg *cw_struct.ClosePosition, ctx sdk.Context,
) (
	sdkResp *perptypes.MsgClosePositionResponse, err error,
) {
	if cwMsg == nil {
		return sdkResp, wasmvmtypes.InvalidRequest{Err: "null close position msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return sdkResp, err
	}

	sdkMsg := &perptypes.MsgClosePosition{
		Sender: cwMsg.Sender,
		Pair:   pair,
	}

	goCtx := sdk.WrapSDKContext(ctx)
	return exec.MsgServer().ClosePosition(goCtx, sdkMsg)
}

func (exec *ExecutorPerp) AddMargin(
	cwMsg *cw_struct.AddMargin, ctx sdk.Context,
) (
	sdkResp *perptypes.MsgAddMarginResponse, err error,
) {
	if cwMsg == nil {
		return sdkResp, wasmvmtypes.InvalidRequest{Err: "null add margin msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return sdkResp, err
	}

	sdkMsg := &perptypes.MsgAddMargin{
		Sender: cwMsg.Sender,
		Pair:   pair,
		Margin: cwMsg.Margin,
	}

	goCtx := sdk.WrapSDKContext(ctx)
	return exec.MsgServer().AddMargin(goCtx, sdkMsg)
}

func (exec *ExecutorPerp) RemoveMargin(
	cwMsg *cw_struct.RemoveMargin, ctx sdk.Context,
) (
	sdkResp *perptypes.MsgRemoveMarginResponse, err error,
) {
	if cwMsg == nil {
		return sdkResp, wasmvmtypes.InvalidRequest{Err: "null remove margin msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return sdkResp, err
	}

	sdkMsg := &perptypes.MsgRemoveMargin{
		Sender: cwMsg.Sender,
		Pair:   pair,
		Margin: cwMsg.Margin,
	}

	goCtx := sdk.WrapSDKContext(ctx)
	return exec.MsgServer().RemoveMargin(goCtx, sdkMsg)
}

func (exec *ExecutorPerp) PegShift(
	cwMsg *cw_struct.PegShift, contractAddr sdk.AccAddress, ctx sdk.Context,
) (err error) {
	if cwMsg == nil {
		return wasmvmtypes.InvalidRequest{Err: "null peg shift msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return err
	}

	return exec.Perp.EditPoolPegMultiplier(
		ctx,
		contractAddr,
		pair,
		cwMsg.PegMult,
	)
}

func (exec *ExecutorPerp) DepthShift(
	cwMsg *cw_struct.DepthShift, contractAddr sdk.AccAddress, ctx sdk.Context,
) (err error) {
	if cwMsg == nil {
		return wasmvmtypes.InvalidRequest{Err: "null pool swap invariant multiplier msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return err
	}

	return exec.Perp.EditPoolSwapInvariant(
		ctx,
		contractAddr,
		pair,
		cwMsg.DepthMult,
	)
}
