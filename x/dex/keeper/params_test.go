package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	app, ctx := testutil.NewNibiruApp(true)

	params := types.DefaultParams()
	app.DexKeeper.SetParams(ctx, params)

	require.EqualValues(t, params, app.DexKeeper.GetParams(ctx))
}
