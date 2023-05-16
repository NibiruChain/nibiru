package keeper

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	types "github.com/NibiruChain/nibiru/x/perp/v1/types"
)

func TestWithdraw(t *testing.T) {
	tests := []struct {
		name                  string
		initialVaultBalance   int64
		initialPrepaidBadDebt int64
		amountToWithdraw      int64

		expectedPerpEFWithdrawal    int64
		expectedFinalPrepaidBadDebt int64
	}{
		{
			name:                  "no bad debt",
			initialVaultBalance:   100,
			initialPrepaidBadDebt: 0,

			amountToWithdraw: 10,

			expectedPerpEFWithdrawal:    0,
			expectedFinalPrepaidBadDebt: 0,
		},
		{
			name:                  "creates prepaid bad debt",
			initialVaultBalance:   9,
			initialPrepaidBadDebt: 0,

			amountToWithdraw: 10,

			expectedPerpEFWithdrawal:    1,
			expectedFinalPrepaidBadDebt: 1,
		},
		{
			name:                  "increases existing prepaid bad debt",
			initialVaultBalance:   9,
			initialPrepaidBadDebt: 1,

			amountToWithdraw: 10,

			expectedPerpEFWithdrawal:    1,
			expectedFinalPrepaidBadDebt: 2,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("initialize variables")
			perpKeeper, mocks, ctx := getKeeper(t)
			receiver := testutil.AccAddress()
			denom := "NUSD"

			t.Log("mock account keeper")
			vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
			mocks.mockAccountKeeper.EXPECT().GetModuleAddress(
				types.VaultModuleAccount).
				Return(vaultAddr)

			t.Log("mock bank keeper")
			mocks.mockBankKeeper.EXPECT().GetBalance(ctx, vaultAddr, denom).
				Return(sdk.NewInt64Coin(denom, tc.initialVaultBalance))
			mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
				ctx, types.VaultModuleAccount, receiver,
				sdk.NewCoins(sdk.NewInt64Coin(denom, tc.amountToWithdraw)),
			).Return(nil)
			if tc.expectedPerpEFWithdrawal > 0 {
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.PerpEFModuleAccount, types.VaultModuleAccount,
					sdk.NewCoins(sdk.NewInt64Coin(denom, tc.expectedPerpEFWithdrawal)),
				).Return(nil)
			}

			t.Log("initial prepaid bad debt")
			perpKeeper.PrepaidBadDebt.Insert(ctx, denom, types.PrepaidBadDebt{Denom: denom, Amount: sdk.NewInt(tc.initialPrepaidBadDebt)})

			t.Log("execute withdrawal")
			err := perpKeeper.Withdraw(ctx, denom, receiver, sdk.NewInt(tc.amountToWithdraw))
			require.NoError(t, err)

			t.Log("assert new prepaid bad debt")
			prepaidBadDebt, err := perpKeeper.PrepaidBadDebt.Get(ctx, denom)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedFinalPrepaidBadDebt, prepaidBadDebt.Amount.Int64())
		})
	}
}

func TestRealizeBadDebt(t *testing.T) {
	tests := []struct {
		name                  string
		initialPrepaidBadDebt int64

		badDebtToRealize int64

		expectedPerpEFWithdrawal    int64
		expectedFinalPrepaidBadDebt int64
	}{
		{
			name:                  "prepaid bad debt completely covers bad debt to realize",
			initialPrepaidBadDebt: 10,

			badDebtToRealize: 5,

			expectedPerpEFWithdrawal:    0,
			expectedFinalPrepaidBadDebt: 5,
		},
		{
			name:                  "prepaid bad debt exactly covers bad debt to realize",
			initialPrepaidBadDebt: 10,

			badDebtToRealize: 10,

			expectedPerpEFWithdrawal:    0,
			expectedFinalPrepaidBadDebt: 0,
		},
		{
			name:                  "requires perpEF withdrawal",
			initialPrepaidBadDebt: 5,

			badDebtToRealize: 10,

			expectedPerpEFWithdrawal:    5,
			expectedFinalPrepaidBadDebt: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("initialize variables")
			perpKeeper, mocks, ctx := getKeeper(t)
			denom := "NUSD"

			if tc.expectedPerpEFWithdrawal > 0 {
				t.Log("mock bank keeper")
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.PerpEFModuleAccount, types.VaultModuleAccount,
					sdk.NewCoins(sdk.NewInt64Coin(denom, tc.expectedPerpEFWithdrawal)),
				).Return(nil)
			}

			t.Log("initial prepaid bad debt")
			perpKeeper.PrepaidBadDebt.Insert(ctx, denom, types.PrepaidBadDebt{
				Denom:  denom,
				Amount: sdk.NewInt(tc.initialPrepaidBadDebt),
			})

			t.Log("execute withdrawal")
			err := perpKeeper.realizeBadDebt(ctx, denom, sdk.NewInt(tc.badDebtToRealize))
			require.NoError(t, err)

			t.Log("assert new prepaid bad debt")
			prepaidBadDebt, err := perpKeeper.PrepaidBadDebt.Get(ctx, denom)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedFinalPrepaidBadDebt, prepaidBadDebt.Amount.Int64())
		})
	}
}

func TestIncrementDecrementBadDebt(t *testing.T) {
	k, _, ctx := getKeeper(t)
	// increment on non-existing prepaid bad debt
	bd := k.IncrementPrepaidBadDebt(ctx, "unibi", sdk.NewInt(1000))
	require.Equal(t, sdk.NewInt(1000), bd)
	// increment on existing
	bd = k.IncrementPrepaidBadDebt(ctx, "unibi", sdk.NewInt(1000))
	require.Equal(t, sdk.NewInt(2000), bd)
	// decrement
	bd = k.DecrementPrepaidBadDebt(ctx, "unibi", sdk.NewInt(1000))
	require.Equal(t, sdk.NewInt(1000), bd)
	// decrement below zero
	bd = k.DecrementPrepaidBadDebt(ctx, "unibi", sdk.NewInt(2000))
	require.Equal(t, sdk.ZeroInt(), bd)
}
