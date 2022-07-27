package antedecorators

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	pricefeedkeeper "github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
)

type GaslessDecorator struct {
	wrapped         []sdk.AnteDecorator
	pricefeedKeeper pricefeedkeeper.Keeper
	perpKeeper      perpkeeper.Keeper
}

func NewGaslessDecorator(wrapped []sdk.AnteDecorator, pricefeedKeeper pricefeedkeeper.Keeper, perpKeeper perpkeeper.Keeper) GaslessDecorator {
	return GaslessDecorator{wrapped: wrapped, pricefeedKeeper: pricefeedKeeper, perpKeeper: perpKeeper}
}

func (gd GaslessDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !isTxGasless(tx, ctx, gd.pricefeedKeeper, gd.perpKeeper) {
		// if not gasless, then we use the wrappers

		// AnteHandle always takes a `next` so we need a no-op to execute only one handler at a time
		terminatorHandler := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			return ctx, nil
		}
		// iterating instead of recursing the handler for readability
		// we use blank here because we shouldn't handle the error
		for _, handler := range gd.wrapped {
			ctx, _ = handler.AnteHandle(ctx, tx, simulate, terminatorHandler)
		}
		return next(ctx, tx, simulate)
	}
	gaslessMeter := sdk.NewInfiniteGasMeter()

	return next(ctx.WithGasMeter(gaslessMeter), tx, simulate)
}

func isTxGasless(tx sdk.Tx, ctx sdk.Context, pricefeedKeeper pricefeedkeeper.Keeper, perpKeeper perpkeeper.Keeper) bool {
	if len(tx.GetMsgs()) == 0 {
		// empty TX shouldn't be gasless
		return false
	}
	for _, msg := range tx.GetMsgs() {
		switch m := msg.(type) {
		case *pricefeedtypes.MsgPostPrice:
			if pricefeedPostPriceIsGasless(m, ctx, pricefeedKeeper) {
				continue
			}
			return false
		case *perptypes.MsgLiquidate:
			if liquidateIsGasless(m, ctx, perpKeeper) {
				continue
			}
			return false
		default:
			return false
		}
	}
	return true
}

func pricefeedPostPriceIsGasless(msg *pricefeedtypes.MsgPostPrice, ctx sdk.Context, keeper pricefeedkeeper.Keeper) bool {
	valAddr, err := sdk.AccAddressFromBech32(msg.Oracle)
	if err != nil {
		return false
	}

	pair := common.AssetPair{Token0: msg.Token0, Token1: msg.Token1}
	fmt.Println(msg.Oracle, keeper.IsWhitelistedOracle(ctx, pair.String(), valAddr))
	return keeper.IsWhitelistedOracle(ctx, pair.String(), valAddr)
}

func liquidateIsGasless(msg *perptypes.MsgLiquidate, ctx sdk.Context, keeper perpkeeper.Keeper) bool {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	return err == nil // TODO: check if within whitelist for liquidators
}
