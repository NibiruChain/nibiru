package app

import (
	"github.com/spf13/cast"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	servertypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/upgrade"
	upgradekeeper "github.com/NibiruChain/nibiru/v2/x/upgrade/keeper"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// initUpgradeKeeper constructs the upgrade keeper before IBC consumes it.
// InitChainer supplies the complete module version map after all modules are
// registered, so this keeper does not need the depinject-only init version map.
func (app *NibiruApp) initUpgradeKeeper(appOpts servertypes.AppOptions) {
	skipUpgradeHeights := make(map[int64]bool)
	var homePath string
	if appOpts != nil {
		for _, height := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
			skipUpgradeHeights[int64(height)] = true
		}
		homePath = cast.ToString(appOpts.Get(flags.FlagHome))
	}

	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		app.keys[upgradetypes.StoreKey],
		app.appCodec,
		homePath,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.UpgradeKeeper.BindWasm(upgradeContractBridge{Keeper: &app.WasmKeeper}, app.SudoKeeper, upgrade.ContractWasm)
}

// upgradeContractBridge selects the permissions of native host calls. The
// Rust contract still checks governance execute callers and the live sudo root.
type upgradeContractBridge struct{ *wasmkeeper.Keeper }

var _ upgradekeeper.ContractBridge = upgradeContractBridge{}

func (b upgradeContractBridge) Execute(ctx sdk.Context, addr, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error) {
	return wasmkeeper.NewDefaultPermissionKeeper(b.Keeper).Execute(ctx, addr, caller, msg, coins)
}
func (b upgradeContractBridge) CreateCode(ctx sdk.Context, creator sdk.AccAddress, code []byte, access *wasmtypes.AccessConfig) (uint64, []byte, error) {
	return wasmkeeper.NewGovPermissionKeeper(b.Keeper).Create(ctx, creator, code, access)
}
func (b upgradeContractBridge) Instantiate2(ctx sdk.Context, codeID uint64, creator, admin sdk.AccAddress, msg []byte, label string, coins sdk.Coins, salt []byte, fixMsg bool) (sdk.AccAddress, []byte, error) {
	return wasmkeeper.NewGovPermissionKeeper(b.Keeper).Instantiate2(ctx, codeID, creator, admin, msg, label, coins, salt, fixMsg)
}
func (upgradeContractBridge) PredictContractAddress(checksum []byte, creator sdk.AccAddress, salt []byte) sdk.AccAddress {
	return wasmkeeper.BuildContractAddressPredictable(checksum, creator, salt, []byte{})
}
