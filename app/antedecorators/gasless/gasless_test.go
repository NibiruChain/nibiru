package gasless_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	gaslessante "github.com/NibiruChain/nibiru/app/antedecorators/gasless"

	pricefeedkeeper "github.com/NibiruChain/nibiru/x/pricefeed/keeper"
)

var output = ""

type FakeAnteDecoratorOne struct{}

func (ad FakeAnteDecoratorOne) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	output = fmt.Sprintf("%sone", output)
	return next(ctx, tx, simulate)
}

type FakeAnteDecoratorTwo struct{}

func (ad FakeAnteDecoratorTwo) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	output = fmt.Sprintf("%stwo", output)
	return next(ctx, tx, simulate)
}

type FakeAnteDecoratorThree struct{}

func (ad FakeAnteDecoratorThree) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	output = fmt.Sprintf("%sthree", output)
	return next(ctx, tx, simulate)
}

type FakeTx struct{}

func (tx FakeTx) GetMsgs() []sdk.Msg {
	return []sdk.Msg{}
}

func (tx FakeTx) ValidateBasic() error {
	return nil
}

func TestGaslessDecorator(t *testing.T) {
	anteDecorators := []sdk.AnteDecorator{
		FakeAnteDecoratorOne{},
		gaslessante.NewGaslessDecorator([]sdk.AnteDecorator{FakeAnteDecoratorTwo{}}, pricefeedkeeper.Keeper{}),
		FakeAnteDecoratorThree{},
	}
	chainedHandler := sdk.ChainAnteDecorators(anteDecorators...)
	_, err := chainedHandler(sdk.Context{}, FakeTx{}, false)
	require.NoError(t, err)
	require.Equal(t, "onetwothree", output)
}
