package upgrades

import (
	"fmt"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
)

var Upgrade2_24_0 = Upgrade{UpgradeName: "v2.24.0", Handler: Handler_v2_24{}}

// Handler_v2_24 runs module migrations, then clears the state of the retired
// x/devgas module. The v2.24 binary no longer pays fee shares or accepts
// x/devgas transaction messages, so the stored fee shares and the enabled
// param have no effect. Clearing them keeps queries and genesis export
// consistent with that behavior.
//
// The upgrade keeps the x/devgas store key. Deleting the store is a later
// upgrade.
type Handler_v2_24 struct{}

var _ HandlerImpl = Handler_v2_24{}

// Handler returns migration errors because the binary requires the migrated
// schema. The x/devgas cleanup is not required for the chain to run, so its
// failure is reported with an upgrade_failure event and does not halt the
// upgrade.
func (h Handler_v2_24) Handler(mm *module.Manager, cfg module.Configurator, nibiru *keepers.PublicKeepers) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versions, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}
		deleted, err := clearDevgasStateV224(ctx, nibiru)
		if err != nil {
			ctx.Logger().Error("v2.24.0 upgrade failure", "err", err)
			ctx.EventManager().EmitEvent(NewEventUpgradeFailure("v2.24.0", err))
			return versions, nil
		}
		ctx.Logger().Info("v2.24.0 cleared x/devgas state", "fee_shares_deleted", deleted)
		return versions, nil
	}
}

// clearDevgasStateV224 deletes the registered fee shares and disables fee
// sharing on a cached context. It commits only when the whole cleanup
// succeeds, so a failure leaves the x/devgas store as it was.
func clearDevgasStateV224(ctx sdk.Context, nibiru *keepers.PublicKeepers) (deleted int, err error) {
	defer func() {
		if value := recover(); value != nil {
			deleted = 0
			err = fmt.Errorf("clear x/devgas state: panic: %v", value)
		}
	}()
	cached, commit := ctx.CacheContext()
	deleted, err = nibiru.DevGasKeeper.ClearFeeShareState(cached)
	if err != nil {
		return 0, fmt.Errorf("clear x/devgas state: %w", err)
	}
	commit()
	return deleted, nil
}
