package wasmbinding

import (
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	"github.com/NibiruChain/nibiru/x/common/asset"
	perpv2keeper "github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type ExecutorPerp struct {
	PerpV2 perpv2keeper.Keeper
}

func (exec *ExecutorPerp) MsgServer() perpv2types.MsgServer {
	return perpv2keeper.NewMsgServerImpl(exec.PerpV2)
}

func (exec *ExecutorPerp) MarketOrder(
	cwMsg *bindings.MarketOrder, sender sdk.AccAddress, ctx sdk.Context,
) (
	sdkResp *perpv2types.MsgMarketOrderResponse, err error,
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

	sdkMsg := &perpv2types.MsgMarketOrder{
		Sender:               sender.String(),
		Pair:                 pair,
		Side:                 side,
		QuoteAssetAmount:     cwMsg.QuoteAmount,
		Leverage:             cwMsg.Leverage,
		BaseAssetAmountLimit: cwMsg.BaseAmountLimit,
	}
	if err := sdkMsg.ValidateBasic(); err != nil {
		return sdkResp, err
	}

	goCtx := sdk.WrapSDKContext(ctx)
	return exec.MsgServer().MarketOrder(goCtx, sdkMsg)
}

func (exec *ExecutorPerp) ClosePosition(
	cwMsg *bindings.ClosePosition, sender sdk.AccAddress, ctx sdk.Context,
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
		Sender: sender.String(),
		Pair:   pair,
	}
	if err := sdkMsg.ValidateBasic(); err != nil {
		return sdkResp, err
	}

	goCtx := sdk.WrapSDKContext(ctx)
	return exec.MsgServer().ClosePosition(goCtx, sdkMsg)
}

func (exec *ExecutorPerp) AddMargin(
	cwMsg *bindings.AddMargin, sender sdk.AccAddress, ctx sdk.Context,
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
		Sender: sender.String(),
		Pair:   pair,
		Margin: cwMsg.Margin,
	}
	if err := sdkMsg.ValidateBasic(); err != nil {
		return sdkResp, err
	}

	goCtx := sdk.WrapSDKContext(ctx)
	return exec.MsgServer().AddMargin(goCtx, sdkMsg)
}

func (exec *ExecutorPerp) RemoveMargin(
	cwMsg *bindings.RemoveMargin, sender sdk.AccAddress, ctx sdk.Context,
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
		Sender: sender.String(),
		Pair:   pair,
		Margin: cwMsg.Margin,
	}
	if err := sdkMsg.ValidateBasic(); err != nil {
		return sdkResp, err
	}

	goCtx := sdk.WrapSDKContext(ctx)
	return exec.MsgServer().RemoveMargin(goCtx, sdkMsg)
}

func (exec *ExecutorPerp) PegShift(
	cwMsg *bindings.PegShift, contractAddr sdk.AccAddress, ctx sdk.Context,
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

func (exec *ExecutorPerp) DepthShift(cwMsg *bindings.DepthShift, ctx sdk.Context) (err error) {
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
	cwMsg *bindings.InsuranceFundWithdraw, ctx sdk.Context,
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

// TODO: rename to CloseMarket
func (exec *ExecutorPerp) SetMarketEnabled(
	cwMsg *bindings.SetMarketEnabled, ctx sdk.Context,
) (err error) {
	if cwMsg == nil {
		return wasmvmtypes.InvalidRequest{Err: "null msg"}
	}

	pair, err := asset.TryNewPair(cwMsg.Pair)
	if err != nil {
		return err
	}

	return exec.PerpV2.CloseMarket(ctx, pair)
}

func (exec *ExecutorPerp) CreateMarket(
	cwMsg *bindings.CreateMarket, ctx sdk.Context,
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
			MaintenanceMarginRatio:          mp.MaintenanceMarginRatio,
			MaxLeverage:                     mp.MaxLeverage,
			LatestCumulativePremiumFraction: mp.LatestCumulativePremiumFraction,
			ExchangeFeeRatio:                mp.ExchangeFeeRatio,
			EcosystemFundFeeRatio:           mp.EcosystemFundFeeRatio,
			LiquidationFeeRatio:             mp.LiquidationFeeRatio,
			PartialLiquidationRatio:         mp.PartialLiquidationRatio,
			FundingRateEpochId:              mp.FundingRateEpochId,
			MaxFundingRate:                  mp.MaxFundingRate,
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
