package v4_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	moduletestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module/testutil"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/exported"
	v1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/migrations/v1"
	v4 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/migrations/v4"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
)

type mockSubspace struct {
	ps types.Params
}

func newMockSubspace(ps types.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSet(ctx sdk.Context, ps exported.ParamSet) {
	*ps.(*types.Params) = ms.ps
}

func TestMigrate(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(v1.ModuleName)
	tKey := sdk.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)
	store := ctx.KVStore(storeKey)

	legacySubspace := newMockSubspace(types.DefaultParams())
	require.NoError(t, v4.Migrate(ctx, store, legacySubspace, cdc))

	var res types.Params
	bz := store.Get(v4.ParamsKey)
	require.NoError(t, cdc.Unmarshal(bz, &res))
	require.Equal(t, legacySubspace.ps, res)
}
