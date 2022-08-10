package gasless_test

import (
	types2 "github.com/NibiruChain/nibiru/app/antedecorators/types"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	gaslessante "github.com/NibiruChain/nibiru/app/antedecorators/gasless"
)

var oracleAddr = sample.AccAddress()

type DecoratorWithNormalGasMeterCheck struct {
	t *testing.T
}

func (ad DecoratorWithNormalGasMeterCheck) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	require.IsType(ad.t, sdk.NewGasMeter(111), ctx.GasMeter())

	return next(ctx, tx, simulate)
}

type DecoratorWithInfiniteGasMeterCheck struct {
	t *testing.T
}

func (ad DecoratorWithInfiniteGasMeterCheck) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	require.IsType(ad.t, types2.NewInfiniteGasMeter(), ctx.GasMeter())

	return next(ctx, tx, simulate)
}

type FakeTx struct{}

func (tx FakeTx) GetMsgs() []sdk.Msg {
	return []sdk.Msg{
		&types.MsgPostPrice{
			Oracle: oracleAddr.String(),
			Token0: "unibi",
			Token1: "unusd",
		},
	}
}

func (tx FakeTx) ValidateBasic() error {
	return nil
}

func TestGaslessDecorator(t *testing.T) {
	app, ctx := testapp.NewNibiruAppAndContext(true)
	ctx = ctx.WithGasMeter(sdk.NewGasMeter(10000000))

	// Whitelist an oracle address.
	app.PricefeedKeeper.WhitelistOracles(ctx, []sdk.AccAddress{oracleAddr})

	anteDecorators := []sdk.AnteDecorator{
		DecoratorWithNormalGasMeterCheck{t},
		gaslessante.NewGaslessDecorator(app.PricefeedKeeper),
		DecoratorWithInfiniteGasMeterCheck{t},
	}

	chainedHandler := sdk.ChainAnteDecorators(anteDecorators...)

	_, err := chainedHandler(ctx, FakeTx{}, false)
	require.NoError(t, err)
}
