package configurator

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/runtime"
	storetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1beta1"
)

// RegisterGovModule gives SDK module tests explicit governance wiring without
// retaining production depinject providers. Call it from StartupConfig.InitApp
// with the injected dependencies, before stores load or genesis runs. These
// test apps start at the current schema and do not need a legacy params subspace.
func RegisterGovModule(app *runtime.App, cdc codec.Codec, accounts govtypes.AccountKeeper, bank govtypes.BankKeeper, staking govtypes.StakingKeeper) (*keeper.Keeper, error) {
	key := storetypes.NewKVStoreKey(govtypes.StoreKey)
	if err := app.RegisterStores(key); err != nil {
		return nil, err
	}
	k := keeper.NewKeeper(cdc, key, accounts, bank, staking, app.MsgServiceRouter(), govtypes.DefaultConfig(), authtypes.NewModuleAddress(govtypes.ModuleName).String())
	k.SetLegacyRouter(v1beta1.NewRouter().AddRoute(govtypes.RouterKey, v1beta1.ProposalHandler))
	if err := app.RegisterModules(gov.NewAppModule(cdc, k, accounts, bank, nil)); err != nil {
		return nil, err
	}
	return k, nil
}
