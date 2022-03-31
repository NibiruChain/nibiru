package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetAndSetLastLockId(t *testing.T) {
	tests := []struct {
		name   string
		lockId uint64
	}{
		{
			name:   "simple lockId",
			lockId: 1,
		},
		{
			name:   "lockId is zero",
			lockId: 0,
		},
		{
			name:   "lockId is max uint64",
			lockId: 18446744073709551615,
		},
	}

	for _, testcase := range tests {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewMatrixApp()
			app.LockupKeeper.SetLastLockId(ctx, tc.lockId)

			require.Equal(t, tc.lockId, app.LockupKeeper.GetLastLockId(ctx))
		})
	}
}
