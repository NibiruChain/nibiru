package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/spot/types"
)

func TestGetParams(t *testing.T) {
	app, ctx := testapp.NewTestNibiruAppAndContext(true)

	params := types.DefaultParams()
	app.SpotKeeper.SetParams(ctx, params)

	require.EqualValues(t, params, app.SpotKeeper.GetParams(ctx))
}
