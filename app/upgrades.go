package app

import (
	"fmt"

	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/docs/statik"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	crisistypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution/types"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	govv1types "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"
	slashingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

var Upgrades = upgrades.AllUpgrades

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
			u.Handler.Handler(
				app.ModuleManager,
				app.Configurator(),
				&app.PublicKeepers,
			),
		)
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
