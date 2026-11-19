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
type Handler_v2_24 struct {
	// ClearDevgas replaces the x/devgas cleanup in tests. Nil selects keeper
	// method ClearFeeShareState.
	ClearDevgas func(ctx sdk.Context) (deleted int, err error)
}

var _ HandlerImpl = Handler_v2_24{}

// Handler fails the upgrade when the migrations or the x/devgas cleanup
// return an error, and it recovers no panic.
//
// The cleanup writes state, so every node must reach the same result. The
// handler has no cleanup error it can identify as one that every node would
// hit identically: the only error ClearFeeShareState returns is a failed
// delete of a key it just listed, which a consistent store cannot produce.
// Any failure may therefore be local to one node, such as a database read
// error. Continuing past it would leave that node with fee shares that other
// nodes deleted, and a different app hash. Returning the error, or letting
// the panic propagate, aborts the block so the node halts at the upgrade
// height. A failure that all nodes share halts them all the same way.
func (h Handler_v2_24) Handler(mm *module.Manager, cfg module.Configurator, nibiru *keepers.PublicKeepers) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versions, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}
		clearDevgas := nibiru.DevGasKeeper.ClearFeeShareState
		if h.ClearDevgas != nil {
			clearDevgas = h.ClearDevgas
		}
		deleted, err := clearDevgas(ctx)
		if err != nil {
			return nil, fmt.Errorf("clear x/devgas state: %w", err)
		}
		ctx.Logger().Info("v2.24.0 cleared x/devgas state", "fee_shares_deleted", deleted)
		return versions, nil
	}
}
