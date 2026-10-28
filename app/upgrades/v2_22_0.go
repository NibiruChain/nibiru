package upgrades

import (
	"fmt"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
)

// Upgrade2_22_0 ships Nibiru BFT and x-consensus. It runs module migrations,
// then creates the x-consensus contract seeded from the current x/consensus
// params with min_block_commit_duration_ms unset, so the upgrade changes no
// behavior until the x/sudo root sets a value.
var Upgrade2_22_0 = Upgrade{
	UpgradeName: "v2.22.0",
	Handler:     Handler_v2_22{},
}

var _ HandlerImpl = (*Handler_v2_22)(nil)

// Handler_v2_22 is the v2.22.0 upgrade handler. See Upgrade2_22_0.
type Handler_v2_22 struct{}

// Handler returns migration errors, since the binary needs the migrated
// schema. A failure to create x-consensus is logged and reported as an
// upgrade_failure event (the v2.16 pattern) and the upgrade continues: the
// x/consensus store still holds the current params, the x/consensus
// EndBlocker keeps them while no contract exists, and a later upgrade can
// retry, because EnsureXConsensus is idempotent and writes nothing on
// failure.
func (h Handler_v2_22) Handler(
	mm *module.Manager,
	cfg module.Configurator,
	nibiru *keepers.PublicKeepers,
) upgradetypes.UpgradeHandler {
	return func(
		ctx sdk.Context,
		plan upgradetypes.Plan,
		fromVM module.VersionMap,
	) (module.VersionMap, error) {
		versions, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}
		if err := ensureXConsensus(ctx, nibiru); err != nil {
			ctx.Logger().Error("v2.22.0 upgrade failure", "err", err)
			ctx.EventManager().EmitEvent(NewEventUpgradeFailure("v2.22.0", err))
		}
		return versions, nil
	}
}

// ensureXConsensus calls EnsureXConsensus and turns a panic into an error,
// so nothing in contract creation can halt the chain at the upgrade height.
func ensureXConsensus(
	ctx sdk.Context, nibiru *keepers.PublicKeepers,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("create x-consensus: panic: %v", r)
		}
	}()
	return nibiru.XConsensusKeeper.EnsureXConsensus(ctx)
}
