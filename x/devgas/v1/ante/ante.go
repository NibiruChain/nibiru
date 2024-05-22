package ante

import (
	"encoding/json"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	devgastypes "github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

var _ sdk.AnteDecorator = (*DevGasPayoutDecorator)(nil)

// DevGasPayoutDecorator Run his after we already deduct the fee from the
// account with the ante.NewDeductFeeDecorator() decorator. We pull funds from
// the FeeCollector ModuleAccount
type DevGasPayoutDecorator struct {
	bankKeeper   BankKeeper
	devgasKeeper IDevGasKeeper
}

func NewDevGasPayoutDecorator(
	bk BankKeeper, fs IDevGasKeeper,
) DevGasPayoutDecorator {
	return DevGasPayoutDecorator{
		bankKeeper:   bk,
		devgasKeeper: fs,
	}
}

func (a DevGasPayoutDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.ErrTxDecode.Wrap("Tx must be a FeeTx")
	}

	err = a.devGasPayout(
		ctx, feeTx,
	)
	if err != nil {
		return ctx, sdkerrors.ErrInsufficientFunds.Wrap(err.Error())
	}

	return next(ctx, tx, simulate)
}

// devGasPayout takes the total fees and redistributes 50% (or param set) to
// the contract developers provided they opted-in to payments.
func (a DevGasPayoutDecorator) devGasPayout(
	ctx sdk.Context,
	tx sdk.FeeTx,
) error {
	params := a.devgasKeeper.GetParams(ctx)
	if !params.EnableFeeShare {
		return nil
	}

	toPay, err := a.getWithdrawAddressesFromMsgs(ctx, tx.GetMsgs())
	if err != nil {
		return err
	}

	// Do nothing if no one needs payment
	if len(toPay) == 0 {
		return nil
	}

	feesPaidOutput, err := a.settleFeePayments(ctx, toPay, params, tx.GetFee())
	if err != nil {
		return err
	}

	bz, err := json.Marshal(feesPaidOutput)
	if err != nil {
		return devgastypes.ErrFeeSharePayment.Wrapf("failed to marshal feesPaidOutput: %s", err.Error())
	}

	return ctx.EventManager().EmitTypedEvent(
		&devgastypes.EventPayoutDevGas{Payouts: string(bz)},
	)
}

type FeeSharePayoutEventOutput struct {
	WithdrawAddress sdk.AccAddress `json:"withdraw_address"`
	FeesPaid        sdk.Coins      `json:"fees_paid"`
}

// settleFeePayments sends the funds to the contract developers
func (a DevGasPayoutDecorator) settleFeePayments(
	ctx sdk.Context, toPay []sdk.AccAddress, params devgastypes.ModuleParams, totalFees sdk.Coins,
) ([]FeeSharePayoutEventOutput, error) {
	allowedFees := getAllowedFees(params, totalFees)

	numPairs := len(toPay)
	feesPaidOutput := make([]FeeSharePayoutEventOutput, numPairs)
	if numPairs > 0 {
		govPercent := params.DeveloperShares
		splitFees := FeePayLogic(allowedFees, govPercent, numPairs)

		// pay fees evenly between all withdraw addresses
		for i, withdrawAddr := range toPay {
			err := a.bankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.FeeCollectorName, withdrawAddr, splitFees)
			feesPaidOutput[i] = FeeSharePayoutEventOutput{
				WithdrawAddress: withdrawAddr,
				FeesPaid:        splitFees,
			}

			if err != nil {
				return nil, devgastypes.ErrFeeSharePayment.Wrapf("failed to pay allowedFees to contract developer: %s", err.Error())
			}
		}
	}

	return feesPaidOutput, nil
}

// getAllowedFees gets the allowed fees to be paid based on the module
// parameters of x/devgas
func getAllowedFees(params devgastypes.ModuleParams, totalFees sdk.Coins) sdk.Coins {
	// Get only allowed governance fees to be paid (helps for taxes)
	var allowedFees sdk.Coins
	if len(params.AllowedDenoms) == 0 {
		// If empty, we allow all denoms to be used as payment
		allowedFees = totalFees
	} else {
		for _, fee := range totalFees.Sort() {
			for _, allowed := range params.AllowedDenoms {
				if fee.Denom == allowed {
					allowedFees = allowedFees.Add(fee)
				}
			}
		}
	}

	return allowedFees
}

// getWithdrawAddressesFromMsgs returns a list of all contract addresses that
// have opted-in to receiving payments
func (a DevGasPayoutDecorator) getWithdrawAddressesFromMsgs(ctx sdk.Context, msgs []sdk.Msg) ([]sdk.AccAddress, error) {
	toPay := make([]sdk.AccAddress, 0)
	for _, msg := range msgs {
		if _, ok := msg.(*wasmtypes.MsgExecuteContract); ok {
			contractAddr, err := sdk.AccAddressFromBech32(
				msg.(*wasmtypes.MsgExecuteContract).Contract,
			)
			if err != nil {
				return nil, err
			}

			shareData, _ := a.devgasKeeper.GetFeeShare(ctx, contractAddr)

			withdrawAddr := shareData.GetWithdrawerAddr()
			if withdrawAddr != nil && !withdrawAddr.Empty() {
				toPay = append(toPay, withdrawAddr)
			}
		}
	}

	return toPay, nil
}

// FeePayLogic takes the total fees and splits them based on the governance
// params and the number of contracts we are executing on. This returns the
// amount of fees each contract developer should get. tested in ante_test.go
func FeePayLogic(fees sdk.Coins, govPercent math.LegacyDec, numPairs int) sdk.Coins {
	var splitFees sdk.Coins
	for _, c := range fees.Sort() {
		rewardAmount := govPercent.MulInt(c.Amount).QuoInt64(int64(numPairs)).RoundInt()
		if !rewardAmount.IsZero() {
			splitFees = splitFees.Add(sdk.NewCoin(c.Denom, rewardAmount))
		}
	}

	return splitFees
}
