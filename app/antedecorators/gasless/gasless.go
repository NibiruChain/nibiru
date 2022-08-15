package gasless

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	types "github.com/NibiruChain/nibiru/app/antedecorators/types"
	"github.com/NibiruChain/nibiru/x/common"
	pricefeedkeeper "github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
)

type GaslessDecorator struct {
	pricefeedKeeper pricefeedkeeper.Keeper
}

func NewGaslessDecorator(pricefeedKeeper pricefeedkeeper.Keeper) GaslessDecorator {
	return GaslessDecorator{pricefeedKeeper: pricefeedKeeper}
}

func (gd GaslessDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if simulate || !isTxGasless(tx, ctx, gd.pricefeedKeeper) {
		return next(ctx, tx, simulate)
	}

	gaslessMeter := types.GasLessMeter()
	return next(ctx.WithGasMeter(gaslessMeter), tx, simulate)
}

func isTxGasless(tx sdk.Tx, ctx sdk.Context, pricefeedKeeper pricefeedkeeper.Keeper) bool {
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
	return keeper.IsWhitelistedOracle(ctx, pair.String(), valAddr)
}
