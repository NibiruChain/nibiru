package binding

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpv2keeper "github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
)

type ExecutorPerp struct {
	PerpV2 perpv2keeper.Keeper
}

func (exec *ExecutorPerp) MsgServer() perpv2types.MsgServer {
	return perpv2keeper.NewMsgServerImpl(exec.PerpV2)
}

func (exec *ExecutorPerp) OpenPosition(
	cwMsg *cw_struct.OpenPosition, ctx sdk.Context,
) (
	sdkResp *perpv2types.MsgOpenPositionResponse, err error,
) {
	if cwMsg == nil {
		return sdkResp, wasmvmtypes.InvalidRequest{Err: "null open position msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return sdkResp, err
	}

	var side perpv2types.Direction
	if cwMsg.IsLong {
		side = perpv2types.Direction_LONG
	} else {
		side = perpv2types.Direction_SHORT
	}

	sdkMsg := &perpv2types.MsgOpenPosition{
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
	sdkResp *perpv2types.MsgClosePositionResponse, err error,
) {
	if cwMsg == nil {
		return sdkResp, wasmvmtypes.InvalidRequest{Err: "null close position msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return sdkResp, err
	}

	sdkMsg := &perpv2types.MsgClosePosition{
		Sender: cwMsg.Sender,
		Pair:   pair,
	}

	goCtx := sdk.WrapSDKContext(ctx)
	return exec.MsgServer().ClosePosition(goCtx, sdkMsg)
}

func (exec *ExecutorPerp) AddMargin(
	cwMsg *cw_struct.AddMargin, ctx sdk.Context,
) (
	sdkResp *perpv2types.MsgAddMarginResponse, err error,
) {
	if cwMsg == nil {
		return sdkResp, wasmvmtypes.InvalidRequest{Err: "null add margin msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return sdkResp, err
	}

	sdkMsg := &perpv2types.MsgAddMargin{
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
	sdkResp *perpv2types.MsgRemoveMarginResponse, err error,
) {
	if cwMsg == nil {
		return sdkResp, wasmvmtypes.InvalidRequest{Err: "null remove margin msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return sdkResp, err
	}

	sdkMsg := &perpv2types.MsgRemoveMargin{
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
		return wasmvmtypes.InvalidRequest{Err: "null msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return err
	}

	return exec.PerpV2.EditPriceMultiplier(
		ctx,
		// contractAddr,
		pair,
		cwMsg.PegMult,
	)
}

func (exec *ExecutorPerp) DepthShift(cwMsg *cw_struct.DepthShift, ctx sdk.Context) (err error) {
	if cwMsg == nil {
		return wasmvmtypes.InvalidRequest{Err: "null msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return err
	}

	return exec.PerpV2.EditSwapInvariant(ctx, pair, cwMsg.DepthMult)
}

func (exec *ExecutorPerp) InsuranceFundWithdraw(
	cwMsg *cw_struct.InsuranceFundWithdraw, ctx sdk.Context,
) (err error) {
	if cwMsg == nil {
		return wasmvmtypes.InvalidRequest{Err: "null msg"}
	}

	to, err := sdk.AccAddressFromBech32(cwMsg.To)
	if err != nil {
		return err
	}

	return exec.PerpV2.Admin().WithdrawFromInsuranceFund(
		ctx,
		cwMsg.Amount,
		to,
	)
}

func (exec *ExecutorPerp) SetMarketEnabled(
	cwMsg *cw_struct.SetMarketEnabled, ctx sdk.Context,
) (err error) {
	if cwMsg == nil {
		return wasmvmtypes.InvalidRequest{Err: "null msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return err
	}

	return exec.PerpV2.Admin().SetMarketEnabled(ctx, pair, cwMsg.Enabled)
}
