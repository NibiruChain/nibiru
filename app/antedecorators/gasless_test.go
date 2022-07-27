package antedecorators_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app/antedecorators"

	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
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
		antedecorators.NewGaslessDecorator([]sdk.AnteDecorator{FakeAnteDecoratorTwo{}}, pricefeedkeeper.Keeper{}, perpkeeper.Keeper{}),
		FakeAnteDecoratorThree{},
	}
	chainedHandler := sdk.ChainAnteDecorators(anteDecorators...)
	chainedHandler(sdk.Context{}, FakeTx{}, false)
	require.Equal(t, "onetwothree", output)
}
