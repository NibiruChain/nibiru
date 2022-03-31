package keeper_test

import (
	"testing"
	"time"

	"github.com/MatrixDao/matrix/x/lockup/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCreateLock(t *testing.T) {
	tests := []struct {
		name                string
		previousLockId      uint64
		accountInitialFunds sdk.Coins
		ownerAddr           sdk.AccAddress
		coins               sdk.Coins
		duration            time.Duration
		shouldErr           bool
	}{
		{
			name:                "happy path",
			previousLockId:      1,
			accountInitialFunds: sdk.NewCoins(sdk.NewInt64Coin("foo", 100)),
			ownerAddr:           sample.AccAddress(),
			coins:               sdk.NewCoins(sdk.NewInt64Coin("foo", 100)),
			duration:            24 * time.Hour,
			shouldErr:           false,
		},
		{
			name:                "not enough funds",
			previousLockId:      1,
			accountInitialFunds: sdk.NewCoins(sdk.NewInt64Coin("foo", 99)),
			ownerAddr:           sample.AccAddress(),
			coins:               sdk.NewCoins(sdk.NewInt64Coin("foo", 100)),
			duration:            24 * time.Hour,
			shouldErr:           true,
		},
	}

	for _, testcase := range tests {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewMatrixApp()
			app.LockupKeeper.SetLastLockId(ctx, tc.previousLockId)
			simapp.FundAccount(app.BankKeeper, ctx, tc.ownerAddr, tc.accountInitialFunds)

			lock, err := app.LockupKeeper.LockTokens(ctx, tc.ownerAddr, tc.coins, tc.duration)
			if tc.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				require.Equal(t, types.Lock{
					LockId:   tc.previousLockId + 1,
					Owner:    tc.ownerAddr.String(),
					Duration: tc.duration,
					Coins:    tc.coins,
				}, lock)

				require.Equal(t, tc.previousLockId+1, app.LockupKeeper.GetLastLockId(ctx))
			}

		})
	}
}
