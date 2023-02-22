package lockup_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/wasm/lockup"
)

func TestLockMsg(t *testing.T) {
	deployer := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, deployer)

	joiner, amount := JoinPool(t, app, ctx)
	lock := instantiateLockupContract(t, ctx, app, deployer)
	require.NotEmpty(t, lock)

	t.Log("Lock lp tokens of joiner into lockup contract")

	msg := lockup.LockupMsg{
		Lock: &lockup.Lock{
			Blocks: 10,
		},
	}

	err := executeCustom(t, ctx, app, lock, joiner, msg, amount)
	require.NoError(t, err)

	// query the denom and see if it matches
	query := lockup.LockupQuery{
		LocksByDenomUnlockingAfter: &lockup.LocksByDenomUnlockingAfter{
			Denom:          amount.Denom,
			UnlockingAfter: int(ctx.BlockHeight() + 5),
		},
	}
	resp := lockup.LocksByDenomUnlockingAfterResponse{}
	queryCustom(t, ctx, app, lock, query, &resp, false)

	require.Equal(t, len(resp), 1)
}
