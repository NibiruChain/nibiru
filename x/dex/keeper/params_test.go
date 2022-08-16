package keeper_test

import (
	"github.com/NibiruChain/nibiru/simapp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/dex/types"
)

func TestGetParams(t *testing.T) {
	app, ctx := simapp.NewTestNibiruAppAndContext(true)

	params := types.DefaultParams()
	app.DexKeeper.SetParams(ctx, params)

	require.EqualValues(t, params, app.DexKeeper.GetParams(ctx))
}
