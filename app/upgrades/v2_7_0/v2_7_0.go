package v2_7_0

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/eth"
)

const UpgradeName = "v2.7.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
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
			err := AddWnibiToNibiruEvm(nibiru, ctx)
			if err != nil {
				panic(fmt.Errorf("v2.7.0 upgrade failure: %w", err))
			}

			return mm.RunMigrations(ctx, cfg, fromVM)
		}
	},
	StoreUpgrades: storetypes.StoreUpgrades{},
}

// AddWnibiToNibiruEvm adds the canonical WNIBI contract address to the EVM
// module parameters.
func AddWnibiToNibiruEvm(
	keepers *keepers.PublicKeepers,
	ctx sdk.Context,
) error {
	wnibiAddrMainnet := appconst.MAINNET_WNIBI_ADDR
	evmParams := keepers.EvmKeeper.GetParams(ctx)
	evmParams.CanonicalWnibi = eth.EIP55Addr{
		Address: wnibiAddrMainnet,
	}

	err := keepers.EvmKeeper.SetParams(ctx, evmParams)

	return err
}
