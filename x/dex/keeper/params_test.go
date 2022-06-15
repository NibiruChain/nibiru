package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/dex/types"
	testutilapp "github.com/NibiruChain/nibiru/x/testutil/app"
)

func TestGetParams(t *testing.T) {
	app, ctx := testutilapp.NewNibiruApp(true)

	params := types.DefaultParams()
	app.DexKeeper.SetParams(ctx, params)

	require.EqualValues(t, params, app.DexKeeper.GetParams(ctx))
}
