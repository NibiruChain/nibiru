package ante

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func MAX_COMMISSION() sdkmath.LegacyDec { return sdkmath.LegacyMustNewDecFromStr("0.25") }

var _ sdk.AnteDecorator = (*AnteDecoratorStakingCommission)(nil)

// AnteDecoratorStakingCommission: Implements sdk.AnteDecorator, enforcing the
// maximum staking commission for validators on the network.
type AnteDecoratorStakingCommission struct{}

func (a AnteDecoratorStakingCommission) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	for _, msg := range tx.GetMsgs() {
		switch msg := msg.(type) {
		case *stakingtypes.MsgCreateValidator:
			rate := msg.Commission.Rate
			if rate.GT(MAX_COMMISSION()) {
				return ctx, NewErrMaxValidatorCommission(rate)
			}
		case *stakingtypes.MsgEditValidator:
			rate := msg.CommissionRate
			if rate != nil && msg.CommissionRate.GT(MAX_COMMISSION()) {
				return ctx, NewErrMaxValidatorCommission(*rate)
			}
		default:
			continue
		}
	}

	return next(ctx, tx, simulate)
}
