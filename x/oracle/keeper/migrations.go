package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	params := m.keeper.Params.GetOr(ctx, types.DefaultParams())
	// set the new parameter with the old one
	params.TwapLookbackWindowSeconds = uint64(params.TwapLookbackWindow.Seconds())
	m.keeper.Params.Set(ctx, params)
	return nil
}
