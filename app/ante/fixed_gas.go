package ante

import (
	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

const (
	OracleModuleTxGas = 500
	OracleSaiWasmGas  = 0
)

var (
	_ sdk.AnteDecorator = AnteDecoratorEnsureSinglePostPriceMessage{}
	_ sdk.AnteDecorator = AnteDecoratorSaiOracle{}
)

// AnteDecoratorEnsureSinglePostPriceMessage ensures that there is only one
// oracle vote message in the transaction and sets the gas meter to a fixed
// value.
type AnteDecoratorEnsureSinglePostPriceMessage struct{}

func (gd AnteDecoratorEnsureSinglePostPriceMessage) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	hasOracleVoteMsg := false
	hasOraclePreVoteMsg := false

	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		switch msg.(type) {
		case *oracletypes.MsgAggregateExchangeRatePrevote:
			hasOraclePreVoteMsg = true
		case *oracletypes.MsgAggregateExchangeRateVote:
			hasOracleVoteMsg = true
		}
	}

	if hasOracleVoteMsg && hasOraclePreVoteMsg {
		if len(msgs) > 2 {
			return ctx, sdkioerrors.Wrap(ErrOracleAnte, "a transaction cannot have more than a single oracle vote and prevote message")
		}

		ctx = ctx.WithGasMeter(NewFixedGasMeter(OracleModuleTxGas))
	} else if hasOraclePreVoteMsg || hasOracleVoteMsg {
		if len(msgs) > 1 {
			return ctx, sdkioerrors.Wrap(ErrOracleAnte, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages")
		}

		ctx = ctx.WithGasMeter(NewFixedGasMeter(OracleModuleTxGas))
	}

	return next(ctx, tx, simulate)
}

// AnteDecoratorSaiOracle checks for Wasm execute contract calls from a set of
// known senders to the Sai oracle contract(s) and lowers gas costs using a fixed
// gas meter.
type AnteDecoratorSaiOracle struct{}

func (anteDec AnteDecoratorSaiOracle) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	return
}
