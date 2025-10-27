package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	inflationtypes "github.com/NibiruChain/nibiru/v2/x/inflation/types"
)

var (
	// Tests the upgrade process and move to newer version of rocksdb
	Upgrade1_0_1 = Upgrade{
		UpgradeName:          "v1.0.1",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	// Newer version of the Cosmos-SDK
	Upgrade1_0_2 = Upgrade{
		UpgradeName:          "v1.0.2",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	// Newer version of the Cosmos-SDK
	Upgrade1_0_3 = Upgrade{
		UpgradeName:          "v1.0.3",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	Upgrade1_1_0 = Upgrade{
		UpgradeName:          "v1.1.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades: store.StoreUpgrades{
			Added: []string{inflationtypes.ModuleName},
		},
	}

	Upgrade1_2_0 = Upgrade{
		UpgradeName:          "v1.2.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	Upgrade1_3_0 = Upgrade{
		UpgradeName: "v1.3.0",
		CreateUpgradeHandler: func(
			mm *module.Manager,
			cfg module.Configurator,
			nibiru *keepers.PublicKeepers,
			clientKeeper clientkeeper.Keeper,
		) upgradetypes.UpgradeHandler {
			return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
				// set the ICS27 consensus version so InitGenesis is not run
				fromVM[icatypes.ModuleName] = mm.GetVersionMap()[icatypes.ModuleName]

				// create ICS27 Controller submodule params, controller module not enabled.
				controllerParams := icacontrollertypes.Params{
					ControllerEnabled: true,
				}

				// create ICS27 Host submodule params
				hostParams := icahosttypes.Params{
					HostEnabled: true,
					AllowMessages: []string{
						sdk.MsgTypeURL(&banktypes.MsgSend{}),
						sdk.MsgTypeURL(&stakingtypes.MsgDelegate{}),
						sdk.MsgTypeURL(&stakingtypes.MsgUndelegate{}),
						sdk.MsgTypeURL(&stakingtypes.MsgBeginRedelegate{}),
						sdk.MsgTypeURL(&distrtypes.MsgWithdrawDelegatorReward{}),
						sdk.MsgTypeURL(&distrtypes.MsgSetWithdrawAddress{}),
						sdk.MsgTypeURL(&distrtypes.MsgFundCommunityPool{}),
						sdk.MsgTypeURL(&authz.MsgExec{}),
						sdk.MsgTypeURL(&authz.MsgGrant{}),
						sdk.MsgTypeURL(&authz.MsgRevoke{}),
						sdk.MsgTypeURL(&ibctransfertypes.MsgTransfer{}),
					},
				}

				// initialize ICS27 module
				icamodule, correctTypecast := mm.Modules[icatypes.ModuleName].(ica.AppModule)
				if !correctTypecast {
					panic("mm.Modules[icatypes.ModuleName] is not of type ica.AppModule")
				}
				icamodule.InitModule(ctx, controllerParams, hostParams)

				return mm.RunMigrations(ctx, cfg, fromVM)
			}
		},
		StoreUpgrades: store.StoreUpgrades{
			Added: []string{icacontrollertypes.StoreKey, icahosttypes.StoreKey},
		},
	}

	Upgrade1_4_0 = Upgrade{
		UpgradeName:          "v1.4.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}

	Upgrade1_5_0 = Upgrade{
		UpgradeName:          "v1.5.0",
		CreateUpgradeHandler: DefaultUpgradeHandler,
		StoreUpgrades:        store.StoreUpgrades{},
	}
)
