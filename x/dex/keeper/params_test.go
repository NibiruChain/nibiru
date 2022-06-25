package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestGetParams(t *testing.T) {
	app, ctx := testapp.NewNibiruAppAndContext(true)

	params := types.DefaultParams()
	app.DexKeeper.SetParams(ctx, params)

	require.EqualValues(t, params, app.DexKeeper.GetParams(ctx))
}
