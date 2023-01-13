package app

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"log"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// NOTE: When performing upgrades, make sure to keep / register the handlers
// for both the current (n) and the previous (n-1) upgrade name. There is a bug
// in a missing value in a log statement for which the fix is not released
var upgradesList = []string{
	"v0.10.0",
	"v0.17.0",
}

func (app NibiruApp) RegisterUpgradeHandlers() {
	// Upgrades names must be in alphabetical order
	// https://github.com/cosmos/cosmos-sdk/issues/11707
	if !sort.StringsAreSorted(upgradesList) {
		log.Fatal("New upgrades must be appended to 'upgradesList' in alphabetical order")
	}
	upgradeInfo, err := app.upgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	for _, upgradeName := range upgradesList {
		app.upgradeKeeper.SetUpgradeHandler(upgradeName, func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			return app.mm.RunMigrations(ctx, app.configurator, fromVM)
		})

		if upgradeName == "v0.17.0" {
			storeUpgrades := storetypes.StoreUpgrades{Deleted: []string{"pricefeed"}}
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}
}
