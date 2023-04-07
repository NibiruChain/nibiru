package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	vpooltypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
)

// CreateVPoolAction creates a vpool
type CreateVPoolAction struct {
	Pair asset.Pair

	QuoteAssetReserve sdk.Dec
	BaseAssetReserve  sdk.Dec

	VPoolConfig vpooltypes.VpoolConfig

	Bias sdk.Dec
}

func (c CreateVPoolAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.VpoolKeeper.CreatePool(
		ctx,
		c.Pair,
		c.QuoteAssetReserve,
		c.BaseAssetReserve,
		c.VPoolConfig,
		c.Bias,
		sdk.OneDec(),
	)
	if err != nil {
		return ctx, err
	}

	keeper.SetPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
		Pair:                            c.Pair,
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
	})

	return ctx, nil
}

// CreateBaseVpool creates a base vpool with:
// - pair: ubtc:uusdc
// - quote asset reserve: 1000
// - base asset reserve: 100
// - vpool config: default
func CreateBaseVpool() CreateVPoolAction {
	return CreateVPoolAction{
		Pair:              asset.NewPair(denoms.BTC, denoms.USDC),
		QuoteAssetReserve: sdk.NewDec(1000e6),
		BaseAssetReserve:  sdk.NewDec(100e6),
		VPoolConfig: vpooltypes.VpoolConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("1"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("1"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.NewDec(10),
		},
	}
}

// CreateCustomVpool creates a vpool with custom parameters
func CreateCustomVpool(
	pair asset.Pair,
	quoteAssetReserve, baseAssetReserve sdk.Dec,
	vpoolConfig vpooltypes.VpoolConfig,
	bias sdk.Dec,
) action.Action {
	return CreateVPoolAction{
		Pair:              pair,
		QuoteAssetReserve: quoteAssetReserve,
		BaseAssetReserve:  baseAssetReserve,
		VPoolConfig:       vpoolConfig,
		Bias:              bias,
	}
}
