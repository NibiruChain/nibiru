package keeper_test

import (
	"github.com/MatrixDao/matrix/x/lockup/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLockState(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()
	lock := &types.Lock{
		LockId:   0,
		Owner:    sample.AccAddress().String(),
		Duration: 1000 * time.Second,
		EndTime:  ctx.BlockTime(),
		Coins:    sdk.NewCoins(sdk.NewCoin("test", sdk.NewInt(1000))),
	}

	// test create
	app.LockupKeeper.LocksState(ctx).Create(lock)
	// test get
	getLock, err := app.LockupKeeper.LocksState(ctx).Get(lock.LockId)
	require.NoError(t, err)
	require.Equal(t, lock, getLock)
	// test get not found

}
