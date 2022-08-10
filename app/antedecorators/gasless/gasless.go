package gasless

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	types "github.com/NibiruChain/nibiru/app/antedecorators/types"
	"github.com/NibiruChain/nibiru/x/common"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
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
	if simulate || !isTxGasless(tx, ctx, gd.pricefeedKeeper, gd.perpKeeper) {
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
	gaslessMeter := types.NewInfiniteGasMeter()

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
		default:
			return false
		}
	}
	return true
}

// Check if the sender is a whitelisted oracle
func pricefeedPostPriceIsGasless(msg *pricefeedtypes.MsgPostPrice, ctx sdk.Context, keeper pricefeedkeeper.Keeper) bool {
	valAddr, err := sdk.AccAddressFromBech32(msg.Oracle)
	if err != nil {
		return false
	}

	pair := common.AssetPair{Token0: msg.Token0, Token1: msg.Token1}
	fmt.Println(msg.Oracle, keeper.IsWhitelistedOracle(ctx, pair.String(), valAddr))
	return keeper.IsWhitelistedOracle(ctx, pair.String(), valAddr)
}
