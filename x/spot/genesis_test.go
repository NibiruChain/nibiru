package spot_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/spot"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	app, ctx := testapp.NewNibiruTestAppAndContext(true)
	spot.InitGenesis(ctx, app.SpotKeeper, genesisState)
	got := spot.ExportGenesis(ctx, app.SpotKeeper)
	require.NotNil(t, got)

	testutil.Fill(&genesisState)
	testutil.Fill(got)

	require.Equal(t, genesisState, *got)
}
