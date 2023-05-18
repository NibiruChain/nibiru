package binding

import (
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/v1/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/v1/types"
	perpv2keeper "github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
)

type ExecutorPerp struct {
	Perp   perpkeeper.Keeper
	PerpV2 perpv2keeper.Keeper
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
		return wasmvmtypes.InvalidRequest{Err: "null msg"}
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

func (exec *ExecutorPerp) DepthShift(cwMsg *cw_struct.DepthShift, ctx sdk.Context) (err error) {
	if cwMsg == nil {
		return wasmvmtypes.InvalidRequest{Err: "null msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return err
	}

	return exec.Perp.EditPoolSwapInvariant(ctx, pair, cwMsg.DepthMult)
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

func (exec *ExecutorPerp) CreateMarket(
	cwMsg *cw_struct.CreateMarket, ctx sdk.Context,
) (err error) {
	if cwMsg == nil {
		return wasmvmtypes.InvalidRequest{Err: "null msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return err
	}

	var market perpv2types.Market
	if cwMsg.MarketParams == nil {
		market = perpv2types.DefaultMarket(pair)
	} else {
		mp := cwMsg.MarketParams
		market = perpv2types.Market{
			Pair:                            pair,
			Enabled:                         true,
			PriceFluctuationLimitRatio:      mp.PriceFluctuationLimitRatio,
			MaintenanceMarginRatio:          mp.MaintenanceMarginRatio,
			MaxLeverage:                     mp.MaxLeverage,
			LatestCumulativePremiumFraction: mp.LatestCumulativePremiumFraction,
			ExchangeFeeRatio:                mp.ExchangeFeeRatio,
			EcosystemFundFeeRatio:           mp.EcosystemFundFeeRatio,
			LiquidationFeeRatio:             mp.LiquidationFeeRatio,
			PartialLiquidationRatio:         mp.PartialLiquidationRatio,
			FundingRateEpochId:              mp.FundingRateEpochId,
			TwapLookbackWindow:              time.Duration(mp.TwapLookbackWindow.Int64()),
			PrepaidBadDebt:                  sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()),
		}
	}

	return exec.PerpV2.Admin().CreateMarket(ctx, perpv2keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: cwMsg.PegMult,
		SqrtDepth:       cwMsg.SqrtDepth,
		Market:          &market,
	})
}
