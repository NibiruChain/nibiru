package ante

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"

	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

const OracleMessageGas = 500

var _ sdk.AnteDecorator = EnsureSinglePostPriceMessageDecorator{}

// EnsureSinglePostPriceMessageDecorator ensures that there is only one oracle vote message in the transaction
// and sets the gas meter to a fixed value.
type EnsureSinglePostPriceMessageDecorator struct{}

func NewPostPriceFixedPriceDecorator() EnsureSinglePostPriceMessageDecorator {
	return EnsureSinglePostPriceMessageDecorator{}
}

func (gd EnsureSinglePostPriceMessageDecorator) AnteHandle(
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
			return ctx, sdkerrors.Wrap(errors.ErrInvalidRequest, "a transaction cannot have more than a single oracle vote and prevote message")
		}

		ctx = ctx.WithGasMeter(NewFixedGasMeter(OracleMessageGas))
	} else if hasOraclePreVoteMsg || hasOracleVoteMsg {
		if len(msgs) > 1 {
			return ctx, sdkerrors.Wrap(errors.ErrInvalidRequest, "a transaction that includes an oracle vote or prevote message cannot have more than those two messages")
		}

		ctx = ctx.WithGasMeter(NewFixedGasMeter(OracleMessageGas))
	}

	return next(ctx, tx, simulate)
}
