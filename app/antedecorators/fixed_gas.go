package antedecorators

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
)

type FixedPriceGasDecorator struct {
	oracleKeeper oraclekeeper.Keeper
}

func NewFixedPriceGasDecorator(pricefeedKeeper oraclekeeper.Keeper) FixedPriceGasDecorator {
	return FixedPriceGasDecorator{oracleKeeper: pricefeedKeeper}
}

func (gd FixedPriceGasDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	if simulate || !isTxGasless(tx, ctx) {
		return next(ctx, tx, simulate)
	}

	gaslessMeter := types.GasLessMeter()

	return next(ctx.WithGasMeter(gaslessMeter), tx, simulate)
}

func isTxGasless(tx sdk.Tx, ctx sdk.Context) bool {
	if len(tx.GetMsgs()) == 0 {
		// empty TX shouldn't be gasless
		return false
	}
	for _, msg := range tx.GetMsgs() {
		switch _ := msg.(type) {
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
