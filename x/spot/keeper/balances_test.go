package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestCheckBalances(t *testing.T) {
	tests := []struct {
		name string

		// test setup
		userInitialFunds sdk.Coins
		coinsToSpend     sdk.Coins

		// expected results
		expectedError error
	}{
		{
			name: "has enough funds",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			coinsToSpend: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			expectedError: nil,
		},
		{
			name: "not enough user funds",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
			),
			coinsToSpend: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
				sdk.NewInt64Coin(denoms.NUSD, 100),
			),
			expectedError: sdkerrors.ErrInsufficientFunds,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext()

			// fund user account
			sender := testutil.AccAddress()
			require.NoError(t, testapp.FundAccount(app.BankKeeper, ctx, sender, tc.userInitialFunds))

			// swap assets
			err := app.SpotKeeper.CheckEnoughBalances(ctx, tc.coinsToSpend, sender)

			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}

			// check user's final funds did not change
			require.Equal(t,
				tc.userInitialFunds,
				app.BankKeeper.GetAllBalances(ctx, sender),
			)
		})
	}
}
