package wasmbinding

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/wasmbinding"
	"github.com/NibiruChain/nibiru/wasmbinding/bindings"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"

	"github.com/cosmos/cosmos-sdk/simapp"
)

func fundAccount(t *testing.T, ctx sdk.Context, app *app.NibiruApp, addr sdk.AccAddress, coins sdk.Coins) {
	err := simapp.FundAccount(
		app.BankKeeper,
		ctx,
		addr,
		coins,
	)
	require.NoError(t, err)
}

func TestOpenPosition(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := SetupCustomApp(t, actor)

	specs := map[string]struct {
		openPosition *bindings.OpenPosition
		expErr       bool
	}{
		// "valid open-position": {
		// 	openPosition: &bindings.OpenPosition{
		// 		Pair: "",
		// 	},
		// },
		"invalid open-position": {
			openPosition: &bindings.OpenPosition{
				Pair: "",
			},
			expErr: false,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// when
			gotErr := wasmbinding.PerformOpenPosition(&app.PerpKeeper, ctx, actor, spec.openPosition)
			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}
