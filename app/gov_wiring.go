package app

import (
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	govkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
)

// initGovKeeper constructs governance explicitly after the SDK keepers and
// stores are available. The legacy router is attached with the IBC keepers in
// initNonDepinjectKeepers; setupUpgrades installs the legacy parameter key table.
func (app *NibiruApp) initGovKeeper() {
	app.GovKeeper = govkeeper.NewKeeper(
		app.appCodec,
		app.keys[govtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.MsgServiceRouter(),
		govtypes.DefaultConfig(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	// Nibiru has no governance hook providers. Keeper.Hooks supplies the same
	// empty hook chain used when depinject received no GovHooksWrapper values.
}
