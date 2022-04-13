package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetNextLockId(t *testing.T) {
	app, ctx := testutil.NewMatrixApp(true)

	for i := 0; i < 10; i++ {
		require.Equal(t, uint64(i), app.LockupKeeper.GetNextLockId(ctx))
	}
}
