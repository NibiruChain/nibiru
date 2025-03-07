package app

import (
	"fmt"

	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_0_1"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_0_2"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_0_3"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_1_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_2_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_3_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_4_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_5_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_0_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_1_0"
)

var Upgrades = []upgrades.Upgrade{
	v1_0_1.Upgrade,
	v1_0_2.Upgrade,
	v1_0_3.Upgrade,
	v1_1_0.Upgrade,
	v1_2_0.Upgrade,
	v1_3_0.Upgrade,
	v1_4_0.Upgrade,
	v1_5_0.Upgrade,
	v2_0_0.Upgrade,
	v2_1_0.Upgrade,
}

func (app *NibiruApp) setupUpgrades() {
	app.setUpgradeHandlers()
	app.setUpgradeStoreLoaders()
}

func (app *NibiruApp) setUpgradeHandlers() {
	for _, u := range Upgrades {
		app.upgradeKeeper.SetUpgradeHandler(u.UpgradeName, u.CreateUpgradeHandler(app.ModuleManager, app.configurator, app.ibcKeeper.ClientKeeper))
	}
}

func (app *NibiruApp) setUpgradeStoreLoaders() {
	upgradeInfo, err := app.upgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk: %s", err.Error()))
	}

	if app.upgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	for _, u := range Upgrades {
		if upgradeInfo.Name == u.UpgradeName {
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &u.StoreUpgrades))
		}
	}
}
