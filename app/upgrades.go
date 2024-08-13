package app

import (
	"fmt"

	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_1_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_2_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_3_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v1_4_0"
)

var Upgrades = []upgrades.Upgrade{
	v1_1_0.Upgrade,
	v1_2_0.Upgrade,
	v1_3_0.Upgrade,
	v1_4_0.Upgrade,
}

func (app *NibiruApp) setupUpgrades() {
	app.setUpgradeHandlers()
	app.setUpgradeStoreLoaders()
}

func (app *NibiruApp) setUpgradeHandlers() {
	for _, u := range Upgrades {
		app.upgradeKeeper.SetUpgradeHandler(u.UpgradeName, u.CreateUpgradeHandler(app.ModuleManager, app.configurator))
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
