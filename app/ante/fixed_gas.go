package ante

import (
	sdkioerrors "cosmossdk.io/errors"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

const (
	OracleModuleTxGas = 500
	ZeroTxGas         = 0
)

var (
	_ sdk.AnteDecorator = AnteDecEnsureSinglePostPriceMessage{}
	_ sdk.AnteDecorator = AnteDecZeroGasActors{}
)

// AnteDecEnsureSinglePostPriceMessage ensures that there is only one
// oracle vote message in the transaction and sets the gas meter to a fixed
// value.
type AnteDecEnsureSinglePostPriceMessage struct{}

func (anteDec AnteDecEnsureSinglePostPriceMessage) AnteHandle(
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

// AnteDecZeroGasActors checks for Wasm execute contract calls from a set of
// known senders to the whitelisted contract(s), giving those transactions zero
// gas costs using a fixed gas meter.
type AnteDecZeroGasActors struct {
	keepers.PublicKeepers
}

func (anteDec AnteDecZeroGasActors) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	zeroGasActors := anteDec.SudoKeeper.GetZeroGasActors(ctx)
	if len(zeroGasActors.Senders) == 0 || len(zeroGasActors.Contracts) == 0 {
		return next(ctx, tx, simulate)
	}

	zeroGasSenders := set.New(zeroGasActors.Senders...)
	zeroGasContracts := set.New(zeroGasActors.Contracts...)

	for idx, msg := range tx.GetMsgs() {
		if idx == 0 {
			signers := msg.GetSigners()
			if len(signers) == 0 {
				return next(ctx, tx, simulate)
			}
			fromAddr := signers[0]
			if !zeroGasSenders.Has(fromAddr.String()) {
				return next(ctx, tx, simulate)
			}
		}

		msgExec, ok := msg.(*wasm.MsgExecuteContract)
		if !ok {
			return next(ctx, tx, simulate)
		}

		if !zeroGasContracts.Has(msgExec.Contract) {
			return next(ctx, tx, simulate)
		}
	}

	newCtx = ctx.WithGasMeter(NewFixedGasMeter(ZeroTxGas))
	return next(newCtx, tx, simulate)
}
