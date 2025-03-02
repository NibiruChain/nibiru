package v2_1_0

import (
	"context"

	"cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

const UpgradeName = "v2.1.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
	CreateUpgradeHandler: func(mm *module.Manager, cfg module.Configurator, clientKeeper clientkeeper.Keeper) upgradetypes.UpgradeHandler {
		return func(c context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			ctx := sdk.UnwrapSDKContext(c)
			// explicitly update the IBC 02-client params, adding the wasm client type if it is not there
			params := clientKeeper.GetParams(ctx)

			hasWasmClient := false
			for _, client := range params.AllowedClients {
				if client == ibcwasmtypes.Wasm {
					hasWasmClient = true
					break
				}
			}

			if !hasWasmClient {
				params.AllowedClients = append(params.AllowedClients, ibcwasmtypes.Wasm)
				clientKeeper.SetParams(ctx, params)
			}

			return mm.RunMigrations(ctx, cfg, fromVM)
		}
	},
	StoreUpgrades: types.StoreUpgrades{
		Added: []string{ibcwasmtypes.ModuleName},
	},
}
