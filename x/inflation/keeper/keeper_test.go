package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/inflation/types"
)

func init() {
	testapp.EnsureNibiruPrefix()
}

func TestBurn(t *testing.T) {
	testCases := []struct {
		name        string
		sender      sdk.AccAddress
		mintCoin    sdk.Coin
		burnCoin    sdk.Coin
		expectedErr error
	}{
		{
			name:        "pass",
			sender:      testutil.AccAddress(),
			mintCoin:    sdk.NewCoin("unibi", sdk.NewInt(100)),
			burnCoin:    sdk.NewCoin("unibi", sdk.NewInt(100)),
			expectedErr: nil,
		},
		{
			name:        "not enough coins",
			sender:      testutil.AccAddress(),
			mintCoin:    sdk.NewCoin("unibi", sdk.NewInt(100)),
			burnCoin:    sdk.NewCoin("unibi", sdk.NewInt(101)),
			expectedErr: fmt.Errorf("spendable balance 100unibi is smaller than 101unibi: insufficient funds"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

			// mint and send money to the sender
			require.NoError(t,
				nibiruApp.BankKeeper.MintCoins(
					ctx, types.ModuleName, sdk.NewCoins(tc.mintCoin)))
			require.NoError(t,
				nibiruApp.BankKeeper.SendCoinsFromModuleToAccount(
					ctx, types.ModuleName, tc.sender, sdk.NewCoins(tc.mintCoin)),
			)

			// Burn coins
			err := nibiruApp.InflationKeeper.Burn(ctx, sdk.NewCoins(tc.burnCoin), tc.sender)
			if tc.expectedErr != nil {
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
