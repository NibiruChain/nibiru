package keeper_test

import (
	"fmt"
	"github.com/MatrixDao/matrix/app"
	"github.com/MatrixDao/matrix/simapp"
	"github.com/MatrixDao/matrix/x/stablecoin/keeper"
	"github.com/MatrixDao/matrix/x/stablecoin/testutil"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMint_Errors(t *testing.T) {
	account := testutil.AccAddress()

	tests := []struct {
		name          string
		prepare       func(app *app.MatrixApp, ctx sdk.Context)
		expectedError error
	}{
		{
			name:          "it should fail when the user does not have enough balance",
			prepare:       func(app *app.MatrixApp, ctx sdk.Context) {},
			expectedError: fmt.Errorf(""),
		},
		{
			name: "",
			prepare: func(app *app.MatrixApp, ctx sdk.Context) {
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			newApp, ctx := simapp.NewApp()
			msgServer := keeper.NewMsgServerImpl(newApp.StableCoinKeeper)
			goCtx := sdk.WrapSDKContext(ctx)

			_, err := msgServer.Mint(goCtx, &types.MsgMint{
				Creator:    account,
				Collateral: sdk.Coin{},
				Gov:        sdk.Coin{},
			})

			require.EqualError(t, err, sdkerrors.Wrap(types.NoCoinFound, "").Error())
		})
	}
}

func TestMint_HappyPath(t *testing.T) {
}
