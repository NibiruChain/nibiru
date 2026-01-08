package upgrades

import (
	"fmt"
	"slices"

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var (
	Upgrade2_0_0 = Upgrade{
		UpgradeName:          "v2.0.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades: store.StoreUpgrades{
			Added: []string{evm.ModuleName},
		},
	}

	Upgrade2_1_0 = Upgrade{
		UpgradeName: "v2.1.0",
		CreateUpgradeHandler: func(
			mm *module.Manager,
			cfg module.Configurator,
			nibiru *keepers.PublicKeepers,
			clientKeeper clientkeeper.Keeper,
		) upgradetypes.UpgradeHandler {
			return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
				// explicitly update the IBC 02-client params, adding the wasm client type if it is not there
				params := clientKeeper.GetParams(ctx)

				hasWasmClient := slices.Contains(params.AllowedClients, ibcwasmtypes.Wasm)

				if !hasWasmClient {
					params.AllowedClients = append(params.AllowedClients, ibcwasmtypes.Wasm)
					clientKeeper.SetParams(ctx, params)
				}

				return mm.RunMigrations(ctx, cfg, fromVM)
			}
		},
		StoreUpgrades: store.StoreUpgrades{
			Added: []string{ibcwasmtypes.ModuleName},
		},
	}

	Upgrade2_2_0 = Upgrade{
		UpgradeName:          "v2.2.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	Upgrade2_3_0 = Upgrade{
		UpgradeName:          "v2.3.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	Upgrade2_4_0 = Upgrade{
		UpgradeName:          "v2.4.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	Upgrade2_5_0 = Upgrade{
		UpgradeName: "v2.5.0",
		CreateUpgradeHandler: func(
			mm *module.Manager,
			cfg module.Configurator,
			nibiru *keepers.PublicKeepers,
			clientKeeper clientkeeper.Keeper,
		) upgradetypes.UpgradeHandler {
			return func(
				ctx sdk.Context,
				plan upgradetypes.Plan,
				fromVM module.VersionMap,
			) (module.VersionMap, error) {
				err := UpgradeStNibiEvmMetadata(nibiru, ctx, appconst.MAINNET_STNIBI_ADDR)
				if err != nil {
					return fromVM, fmt.Errorf("v2.5.0 upgrade failure: %w", err)
				}

				return mm.RunMigrations(ctx, cfg, fromVM)
			}
		},
		StoreUpgrades: store.StoreUpgrades{},
	}

	Upgrade2_6_0 = Upgrade{
		UpgradeName:          "v2.6.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	Upgrade2_7_0 = Upgrade{
		UpgradeName: "v2.7.0",
		CreateUpgradeHandler: func(
			mm *module.Manager,
			cfg module.Configurator,
			nibiru *keepers.PublicKeepers,
			clientKeeper clientkeeper.Keeper,
		) upgradetypes.UpgradeHandler {
			return func(
				ctx sdk.Context,
				plan upgradetypes.Plan,
				fromVM module.VersionMap,
			) (module.VersionMap, error) {
				err := runUpgrade2_7_0(nibiru, ctx)
				if err != nil {
					return fromVM, fmt.Errorf("v2.7.0 upgrade failure: %w", err)
				}

				return mm.RunMigrations(ctx, cfg, fromVM)
			}
		},
		StoreUpgrades: store.StoreUpgrades{},
	}

	Upgrade2_8_0 = Upgrade{
		UpgradeName:          "v2.8.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	Upgrade2_9_0 = Upgrade{
		UpgradeName:          "v2.9.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	Upgrade2_10_0 = Upgrade{
		UpgradeName:          "v2.10.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}
)
