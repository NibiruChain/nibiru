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
		burnCoin    sdk.Coin
		expectedErr error
	}{
		{
			name:        "pass",
			sender:      testutil.AccAddress(),
			burnCoin:    sdk.NewCoin("nibiru", sdk.NewInt(100)),
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
			nibiruApp.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(tc.burnCoin))
			nibiruApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, tc.sender, sdk.NewCoins(tc.burnCoin))

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
