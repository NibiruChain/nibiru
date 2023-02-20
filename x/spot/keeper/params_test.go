package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

func TestGetParams(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)

	params := types.DefaultParams()
	app.SpotKeeper.SetParams(ctx, params)

	require.EqualValues(t, params, app.SpotKeeper.GetParams(ctx))
}
