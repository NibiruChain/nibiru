package v3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	moduletestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module/testutil"
	paramtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/types"
	v3 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/migrations/v3"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"
)

func TestStoreMigration(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	stakingKey := sdk.NewKVStoreKey("staking")
	tStakingKey := sdk.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(stakingKey, tStakingKey)
	paramstore := paramtypes.NewSubspace(encCfg.Codec, encCfg.Amino, stakingKey, tStakingKey, "staking")

	// Check no params
	require.False(t, paramstore.Has(ctx, types.KeyMinCommissionRate))

	// Run migrations.
	err := v3.MigrateStore(ctx, stakingKey, encCfg.Codec, paramstore)
	require.NoError(t, err)

	// Make sure the new params are set.
	require.True(t, paramstore.Has(ctx, types.KeyMinCommissionRate))
}
