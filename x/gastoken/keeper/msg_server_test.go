package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
)

func TestUpdateFeeToken(t *testing.T) {
	testCases := []struct {
		name              string
		malleate          func(app *app.NibiruApp, ctx sdk.Context)
		msg               types.MsgUpdateFeeToken
		expectedFeeTokens []types.FeeToken
		expectedErr       error
	}{
		{
			name: "successfully add fee token",
			malleate: func(app *app.NibiruApp, ctx sdk.Context) {
				feeToken := validFeeToken
				app.GasTokenKeeper.SetFeeToken(ctx, feeToken)
			},
			msg: types.MsgUpdateFeeToken{
				Sender:   testutil.ADDR_SUDO_ROOT,
				Action:   types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_ADD,
				FeeToken: &anotherValidFeeToken,
			},
			expectedFeeTokens: validFeeTokens,
			expectedErr:       nil,
		},
		{
			name: "successfully remove fee token",
			malleate: func(app *app.NibiruApp, ctx sdk.Context) {
				feeTokens := validFeeTokens
				app.GasTokenKeeper.SetFeeTokens(ctx, feeTokens)
			},
			msg: types.MsgUpdateFeeToken{
				Sender:   testutil.ADDR_SUDO_ROOT,
				Action:   types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_REMOVE,
				FeeToken: &anotherValidFeeToken,
			},
			expectedFeeTokens: []types.FeeToken{validFeeToken},
			expectedErr:       nil,
		},
		{
			name: "fail to add invalid fee token address",
			malleate: func(app *app.NibiruApp, ctx sdk.Context) {
				feeTokens := validFeeTokens
				app.GasTokenKeeper.SetFeeTokens(ctx, feeTokens)
			},
			msg: types.MsgUpdateFeeToken{
				Sender:   testutil.ADDR_SUDO_ROOT,
				Action:   types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_ADD,
				FeeToken: &invalidFeeToken,
			},
			expectedErr: fmt.Errorf("invalid fee token address %s: must be a valid hex address", invalidAddress),
		},
		{
			name: "fail to add an existed fee token address",
			malleate: func(app *app.NibiruApp, ctx sdk.Context) {
				feeTokens := validFeeTokens
				app.GasTokenKeeper.SetFeeTokens(ctx, feeTokens)
			},
			msg: types.MsgUpdateFeeToken{
				Sender:   testutil.ADDR_SUDO_ROOT,
				Action:   types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_ADD,
				FeeToken: &validFeeToken,
			},
			expectedErr: fmt.Errorf("fee token with address %s already exists", validFeeToken.Erc20Address),
		},
		{
			name: "fail to remove an non-existed fee token address",
			malleate: func(app *app.NibiruApp, ctx sdk.Context) {
				feeToken := validFeeToken
				app.GasTokenKeeper.SetFeeToken(ctx, feeToken)
			},
			msg: types.MsgUpdateFeeToken{
				Sender:   testutil.ADDR_SUDO_ROOT,
				Action:   types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_REMOVE,
				FeeToken: &anotherValidFeeToken,
			},
			expectedErr: fmt.Errorf("fee token with address %s not exists", anotherValidFeeToken.Erc20Address),
		},
		{
			name: "fail to remove an invalid fee token address",
			malleate: func(app *app.NibiruApp, ctx sdk.Context) {
				feeTokens := validFeeTokens
				app.GasTokenKeeper.SetFeeTokens(ctx, feeTokens)
			},
			msg: types.MsgUpdateFeeToken{
				Sender:   testutil.ADDR_SUDO_ROOT,
				Action:   types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_REMOVE,
				FeeToken: &invalidFeeToken,
			},
			expectedErr: fmt.Errorf("invalid fee token address %s: must be a valid hex address", invalidFeeToken.Erc20Address),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

			tc.malleate(nibiruApp, ctx)

			if tc.expectedErr != nil {
				_, err := nibiruApp.GasTokenKeeper.UpdateFeeToken(sdk.WrapSDKContext(ctx), &tc.msg)
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErr.Error())
				return
			}

			_, err := nibiruApp.GasTokenKeeper.UpdateFeeToken(sdk.WrapSDKContext(ctx), &tc.msg)
			require.NoError(t, err)

			feeTokens := nibiruApp.GasTokenKeeper.GetFeeTokens(ctx)
			sortFeeTokens(feeTokens)
			sortFeeTokens(tc.expectedFeeTokens)
			require.Equal(t, feeTokens, tc.expectedFeeTokens)
		})
	}
}
