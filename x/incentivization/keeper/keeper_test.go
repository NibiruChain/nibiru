package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestKeeper_CreateIncentivizationProgram(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app, ctx := testutil.NewNibiruApp(false)

		createdProgram, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "denom", 48*time.Hour, ctx.BlockTime(), 1000)
		require.NoError(t, err)

		gotProgram, err := app.IncentivizationKeeper.IncentivizationProgramsState(ctx).Get(0)
		require.NoError(t, err)
		require.Equal(t, createdProgram, gotProgram)
	})
}
