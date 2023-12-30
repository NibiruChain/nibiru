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

	return exec.PerpV2.Sudo().CloseMarket(ctx, pair)
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
			OraclePair:                      asset.MustNewPair(mp.OraclePair),
		}
	}

	return exec.PerpV2.Sudo().CreateMarket(ctx, perpv2keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: cwMsg.PegMult,
		SqrtDepth:       cwMsg.SqrtDepth,
		Market:          &market,
	})
}
