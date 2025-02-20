package ante

import (
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

// VoteFeeDiscountDecorator checks if the Tx signer is a validator and
// has or hasn't voted in the current voting period. If it's their first time,
// we apply a discount on fees or gas. Otherwise, normal cost applies.
//
// In real code, you'll likely store more config, such as a reference to the
// staking keeper to fetch validator info, or track the current epoch.
type VoteFeeDiscountDecorator struct {
	oracleKeeper  OracleKeeperI
	stakingKeeper StakingKeeperI
}

// NewVoteFeeDiscountDecorator is the constructor.
func NewVoteFeeDiscountDecorator(
	oracleKeeper OracleKeeperI,
	stakingKeeper StakingKeeperI,
) VoteFeeDiscountDecorator {
	return VoteFeeDiscountDecorator{
		oracleKeeper:  oracleKeeper,
		stakingKeeper: stakingKeeper,
	}
}

// AnteHandle implements sdk.AnteDecorator. The discount logic is
// purely demonstrative; adapt to your own logic.
func (vfd VoteFeeDiscountDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	// 1. We check if there's exactly one signer (typical for Oracle votes).
	//    If your chain supports multiple signers, adapt accordingly.
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(errortypes.ErrTxDecode, "invalid tx type %T", tx)
	}

	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return next(ctx, tx, simulate)
	}
	if len(sigs) != 1 {
		// passthrough if not exactly one signer
		return next(ctx, tx, simulate)
	}

	// ensure all messages are prevoting or voting messages
	for _, msg := range tx.GetMsgs() {
		if _, ok := msg.(*types.MsgAggregateExchangeRatePrevote); !ok {
			if _, ok := msg.(*types.MsgAggregateExchangeRateVote); !ok {
				return next(ctx, tx, simulate)
			}
		}
	}

	// 2. Check if the signer is a validator
	valAddr := sdk.ValAddress(sigTx.GetSigners()[0])
	validator, found := vfd.stakingKeeper.GetValidator(ctx, valAddr)
	if !found {
		return next(ctx, tx, simulate)
	}

	if validator.Jailed {
		return next(ctx, tx, simulate)
	}

	// needs to have at least 0.5% of the supply bonded
	totalBonded := vfd.stakingKeeper.TotalBondedTokens(ctx)
	currentlyBonded := validator.Tokens

	if currentlyBonded.LT(totalBonded.Mul(math.NewInt(50)).Quo(math.NewInt(10000))) {
		return next(ctx, tx, simulate)
	}

	// 3. If validator, we check whether they've posted a vote this period
	hasVoted := vfd.oracleKeeper.HasVotedInCurrentPeriod(ctx, valAddr)
	if !hasVoted {
		// 4. If first time, let's say we reduce gas cost.
		minGasPrices := ctx.MinGasPrices()
		var discounted []sdk.DecCoin
		for _, mgp := range minGasPrices {
			discounted = append(discounted, sdk.NewDecCoinFromDec(mgp.Denom, mgp.Amount.QuoInt64(69_420)))
		}
		// We'll create a new context with the updated MinGasPrices
		ctx = ctx.WithMinGasPrices(discounted)
	}

	// 5. Keep going in the AnteHandler chain
	return next(ctx, tx, simulate)
}
