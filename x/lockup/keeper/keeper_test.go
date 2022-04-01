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
		accountInitialFunds sdk.Coins
		ownerAddr           sdk.AccAddress
		coins               sdk.Coins
		duration            time.Duration
		shouldErr           bool
	}{
		{
			name:                "happy path",
			accountInitialFunds: sdk.NewCoins(sdk.NewInt64Coin("foo", 100)),
			ownerAddr:           sample.AccAddress(),
			coins:               sdk.NewCoins(sdk.NewInt64Coin("foo", 100)),
			duration:            24 * time.Hour,
			shouldErr:           false,
		},
		{
			name:                "not enough funds",
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
			simapp.FundAccount(app.BankKeeper, ctx, tc.ownerAddr, tc.accountInitialFunds)

			lock, err := app.LockupKeeper.LockTokens(ctx, tc.ownerAddr, tc.coins, tc.duration)
			if tc.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				require.Equal(t, types.Lock{
					LockId:   0,
					Owner:    tc.ownerAddr.String(),
					Duration: tc.duration,
					Coins:    tc.coins,
					EndTime:  ctx.BlockTime().Add(24 * time.Hour),
				}, lock)

				require.Equal(t, uint64(1), app.LockupKeeper.GetNextLockId(ctx))
			}

		})
	}
}
