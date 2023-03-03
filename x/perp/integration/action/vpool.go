package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

// CreateVPoolAction creates a vpool
type CreateVPoolAction struct {
	Pair asset.Pair

	QuoteAssetReserve sdk.Dec
	BaseAssetReserve  sdk.Dec

	VPoolConfig vpooltypes.VpoolConfig
}

func (c CreateVPoolAction) Do(app *app.NibiruApp, ctx sdk.Context) error {
	return app.VpoolKeeper.CreatePool(
		ctx,
		c.Pair,
		sdk.NewDec(1000),
		sdk.NewDec(100),
		vpooltypes.DefaultVpoolConfig(),
	)
}

// CreateBaseVpool creates a base vpool with:
// - pair: ubtc:uusdc
// - quote asset reserve: 1000
// - base asset reserve: 100
// - vpool config: default
func CreateBaseVpool() CreateVPoolAction {
	return CreateVPoolAction{
		Pair:              "ubtc:uusdc",
		QuoteAssetReserve: sdk.NewDec(1000),
		BaseAssetReserve:  sdk.NewDec(100),
		VPoolConfig:       vpooltypes.DefaultVpoolConfig(),
	}
}
