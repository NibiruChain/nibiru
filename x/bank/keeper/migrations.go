package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/v2/x/bank/exported"
	v4 "github.com/NibiruChain/nibiru/v2/x/bank/migrations/v4"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper         BaseKeeper
	legacySubspace exported.Subspace
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper BaseKeeper, legacySubspace exported.Subspace) Migrator {
	return Migrator{keeper: keeper, legacySubspace: legacySubspace}
}

// Migrate3to4 migrates x/bank storage from version 3 to 4.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	m.MigrateSendEnabledParams(ctx)
	return v4.MigrateStore(ctx, m.keeper.storeKey, m.legacySubspace, m.keeper.cdc)
}

// MigrateSendEnabledParams get params from x/params and update the bank params.
// This function is only needed for chains having migrated from <= v0.47 to v0.47.0-5
func (m Migrator) MigrateSendEnabledParams(ctx sdk.Context) {
	sendEnabled := types.GetSendEnabledParams(ctx, m.legacySubspace)
	m.keeper.SetAllSendEnabled(ctx, sendEnabled)
}
