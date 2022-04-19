package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/lockup/keeper"
	"github.com/NibiruChain/nibiru/x/lockup/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLockState(t *testing.T) {
	app, ctx := testutil.NewNibiruApp(true)
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
	getLock, err := app.LockupKeeper.LocksState(ctx).Get(keeper.LockStartID) // we're getting the first starting
	require.NoError(t, err)
	require.Equal(t, lock, getLock)
	// test delete
	err = app.LockupKeeper.LocksState(ctx).Delete(getLock)
	require.NoError(t, err)
	// test get not found
	_, err = app.LockupKeeper.LocksState(ctx).Get(getLock.LockId)
	require.ErrorIs(t, err, types.ErrLockupNotFound)
	// test delete not found
	err = app.LockupKeeper.LocksState(ctx).Delete(lock)
	require.ErrorIs(t, err, types.ErrLockupNotFound)
}
