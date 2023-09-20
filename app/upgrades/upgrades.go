package upgrades

import (
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func emptyUpgradeHandler(mm *module.Manager, configurator module.Configurator, bm BaseAppParamManager) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

type BaseAppParamManager interface {
	GetConsensusParams(ctx sdk.Context) *tmproto.ConsensusParams
	StoreConsensusParams(ctx sdk.Context, cp *tmproto.ConsensusParams)
}

type Upgrade struct {
	UpgradeName          string
	CreateUpgradeHandler func(*module.Manager, module.Configurator, BaseAppParamManager) upgradetypes.UpgradeHandler
	StoreUpgrades        types.StoreUpgrades
}

var Upgrade_0_21_10 = Upgrade{
	UpgradeName:          "0.21.10",
	CreateUpgradeHandler: emptyUpgradeHandler,
	StoreUpgrades:        types.StoreUpgrades{},
}
