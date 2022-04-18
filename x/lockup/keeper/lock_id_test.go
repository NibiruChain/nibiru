package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetNextLockId(t *testing.T) {
	app, ctx := testutil.NewNibiruApp(true)

	for i := 0; i < 10; i++ {
		require.Equal(t, uint64(i), app.LockupKeeper.GetNextLockId(ctx))
	}
}
