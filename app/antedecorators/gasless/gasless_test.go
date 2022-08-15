package gasless_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	types3 "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	gaslessante "github.com/NibiruChain/nibiru/app/antedecorators/gasless"
	types2 "github.com/NibiruChain/nibiru/app/antedecorators/types"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
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
	require.IsType(ad.t, types2.GasLessMeter(), ctx.GasMeter())

	return next(ctx, tx, simulate)
}

type TxWithPostPriceMsg struct{}

func (tx TxWithPostPriceMsg) GetMsgs() []sdk.Msg {
	return []sdk.Msg{
		&types.MsgPostPrice{
			Oracle: oracleAddr.String(),
			Token0: "unibi",
			Token1: "unusd",
		},
	}
}

func (tx TxWithPostPriceMsg) ValidateBasic() error {
	return nil
}

type TxWithoutPostPriceMsg struct{}

func (tx TxWithoutPostPriceMsg) GetMsgs() []sdk.Msg {
	return []sdk.Msg{
		&types3.MsgSend{},
	}
}

func (tx TxWithoutPostPriceMsg) ValidateBasic() error {
	return nil
}

func TestGaslessDecorator_Whitelisted(t *testing.T) {
	tests := []struct {
		name              string
		isWhitelisted     bool
		shouldChangeMeter bool
		tx                sdk.Tx
		simulate          bool
	}{
		{
			/* name */ "whitelisted address",
			/* isWhitelisted */ true,
			/* shouldChangeMeter */ true,
			/* tx */ TxWithPostPriceMsg{},
			/* simulate */ false,
		},
		{
			/* name */ "whitelisted address, simulation",
			/* isWhitelisted */ true,
			/* shouldChangeMeter */ false,
			/* tx */ TxWithPostPriceMsg{},
			/* simulate */ true,
		},
		{
			/* name */ "whitelisted address but tx without price feed message",
			/* isWhitelisted */ true,
			/* shouldChangeMeter */ false,
			/* tx */ TxWithoutPostPriceMsg{},
			/* simulate */ false,
		},
		{
			/* name */ "not whitelisted address with post price tx",
			/* isWhitelisted */ false,
			/* shouldChangeMeter */ false,
			/* tx */ TxWithPostPriceMsg{},
			/* simulate */ false,
		},
		{
			/* name */ "not whitelisted address without post price tx",
			/* isWhitelisted */ false,
			/* shouldChangeMeter */ false,
			/* tx */ TxWithoutPostPriceMsg{},
			/* simulate */ false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruAppAndContext(true)
			ctx = ctx.WithGasMeter(sdk.NewGasMeter(10000000))

			if tc.isWhitelisted {
				// If we whitelist, the gas meter changes.
				app.PricefeedKeeper.WhitelistOracles(ctx, []sdk.AccAddress{oracleAddr})
			}

			var anteDecorators []sdk.AnteDecorator
			if tc.shouldChangeMeter {
				anteDecorators = []sdk.AnteDecorator{
					DecoratorWithNormalGasMeterCheck{t},
					gaslessante.NewGaslessDecorator(app.PricefeedKeeper),
					DecoratorWithInfiniteGasMeterCheck{t},
				}
			} else {
				anteDecorators = []sdk.AnteDecorator{
					DecoratorWithNormalGasMeterCheck{t},
					gaslessante.NewGaslessDecorator(app.PricefeedKeeper),
					DecoratorWithNormalGasMeterCheck{t},
				}
			}

			chainedHandler := sdk.ChainAnteDecorators(anteDecorators...)

			_, err := chainedHandler(ctx, tc.tx, tc.simulate)
			require.NoError(t, err)
		})
	}
}
