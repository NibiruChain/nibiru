package app

import (
	"fmt"

	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_2_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_3_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_4_0"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_5_0"
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
	v2_2_0.Upgrade,
	v2_3_0.Upgrade,
	v2_4_0.Upgrade,
	v2_5_0.Upgrade,
}

func (app *NibiruApp) setupUpgrades() {
	// see https://github.com/cosmos/cosmos-sdk/blob/666c345ad23ddda9523cc5cd1b71187d91c26f34/simapp/upgrades.go#L35-L57
	for _, subspace := range app.paramsKeeper.GetSubspaces() {
		switch subspace.Name() {
		case authtypes.ModuleName:
			subspace.WithKeyTable(authtypes.ParamKeyTable()) //nolint:staticcheck
		case banktypes.ModuleName:
			subspace.WithKeyTable(banktypes.ParamKeyTable()) //nolint:staticcheck
		case stakingtypes.ModuleName:
			subspace.WithKeyTable(stakingtypes.ParamKeyTable()) //nolint:staticcheck
		case distrtypes.ModuleName:
			subspace.WithKeyTable(distrtypes.ParamKeyTable()) //nolint:staticcheck
		case slashingtypes.ModuleName:
			subspace.WithKeyTable(slashingtypes.ParamKeyTable()) //nolint:staticcheck
		case govtypes.ModuleName:
			subspace.WithKeyTable(govv1types.ParamKeyTable()) //nolint:staticcheck
		case crisistypes.ModuleName:
			subspace.WithKeyTable(crisistypes.ParamKeyTable()) //nolint:staticcheck
		}
	}

	app.setUpgradeHandlers()
	app.setUpgradeStoreLoaders()
}

func (app *NibiruApp) setUpgradeHandlers() {
	for _, u := range Upgrades {
		app.UpgradeKeeper.SetUpgradeHandler(u.UpgradeName,
			u.CreateUpgradeHandler(
				app.ModuleManager,
				app.Configurator(),
				&app.PublicKeepers,
				app.ibcKeeper.ClientKeeper,
			))
	}
}

func (app *NibiruApp) setUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk: %s", err.Error()))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	for _, u := range Upgrades {
		if upgradeInfo.Name == u.UpgradeName {
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &u.StoreUpgrades))
		}
	}
}
