package fee

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/app/antedecorators/types"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

const OracleMessageGas = 500

// EnsureSinglePostPriceMessageDecorator ensures that there is only one oracle vote message in the transaction
// and sets the gas meter to a fixed value.
type EnsureSinglePostPriceMessageDecorator struct {
}

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
	for _, msg := range tx.GetMsgs() {
		switch msg.(type) {
		case *oracletypes.MsgAggregateExchangeRatePrevote:
			hasOracleVoteMsg = true
		case *oracletypes.MsgAggregateExchangeRateVote:
			hasOracleVoteMsg = true
		}
	}

	if hasOracleVoteMsg {
		if len(tx.GetMsgs()) > 1 {
			return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "oracle vote message must be the only message in the transaction")
		}

		ctx = ctx.WithGasMeter(types.NewFixedGasMeter(OracleMessageGas))
	}

	return next(ctx, tx, simulate)
}
