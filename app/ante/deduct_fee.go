package ante

import (
	"fmt"
	"math"

	sdkmath "cosmossdk.io/math"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
)

// DeductFeeDecorator deducts fees from the fee payer. The fee payer is the fee
// granter (if specified) or first signer of the tx. If the fee payer does not
// have the funds to pay for the fees, return an InsufficientFunds error.
// Call next AnteHandler if fees successfully deducted.
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type DeductFeeDecorator struct {
	accountKeeper  authante.AccountKeeper
	bankKeeper     types.BankKeeper
	feegrantKeeper authante.FeegrantKeeper
	txFeeChecker   authante.TxFeeChecker
}

func NewDeductFeeDecorator(ak authante.AccountKeeper, bk types.BankKeeper, fk authante.FeegrantKeeper, tfc authante.TxFeeChecker) DeductFeeDecorator {
	if tfc == nil {
		tfc = checkTxFeeWithValidatorMinGasPrices
	}

	return DeductFeeDecorator{
		accountKeeper:  ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		txFeeChecker:   tfc,
	}
}

func (anteDec DeductFeeDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	var (
		priority  int64
		err       error
		feeTx, ok = tx.(sdk.FeeTx)
	)

	// Early return in the case of zero gas meter, skipping fee deduction.
	if isZeroGasMeter(ctx) {
		newCtx := ctx.WithPriority(priority)
		return next(newCtx, tx, simulate)
	}

	if !ok {
		return ctx, sdkioerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, sdkioerrors.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	// Deduct fees
	fee := feeTx.GetFee()
	if !simulate {
		fee, priority, err = anteDec.txFeeChecker(ctx, tx)
		if err != nil {
			return ctx, err
		}
	}
	if err := anteDec.checkDeductFee(ctx, tx, fee); err != nil {
		return ctx, err
	}

	newCtx := ctx.WithPriority(priority)
	return next(newCtx, tx, simulate)
}

func (anteDec DeductFeeDecorator) checkDeductFee(
	ctx sdk.Context,
	sdkTx sdk.Tx,
	fee sdk.Coins,
) error {
	feeTx, ok := sdkTx.(sdk.FeeTx)
	if !ok {
		return sdkioerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if addr := anteDec.accountKeeper.GetModuleAddress(types.FeeCollectorName); addr == nil {
		return fmt.Errorf("fee collector module account (%s) has not been set", types.FeeCollectorName)
	}

	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	deductFeesFrom := feePayer

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if anteDec.feegrantKeeper == nil {
			return sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := anteDec.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, sdkTx.GetMsgs())
			if err != nil {
				return sdkioerrors.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := anteDec.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	// deduct the fees
	if !fee.IsZero() {
		err := deductFees(anteDec.bankKeeper, ctx, deductFeesFromAcc, fee)
		if err != nil {
			return err
		}
	}

	events := sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, fee.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, deductFeesFrom.String()),
		),
	}
	ctx.EventManager().EmitEvents(events)

	return nil
}

// deductFees deducts fees from the given account.
func deductFees(
	bankKeeper types.BankKeeper,
	ctx sdk.Context,
	acc types.AccountI,
	fees sdk.Coins,
) error {
	if !fees.IsValid() {
		return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
	if err != nil {
		return sdkioerrors.Wrap(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	return nil
}

// isZeroGasMeter checks if the context has a zero gas meter set.
// ZeroGas is detected in fixed_gas.go and sets NewFixedGasMeter(ZeroTxGas) to ctx
func isZeroGasMeter(ctx sdk.Context) bool {
	gasMeter := ctx.GasMeter()
	if _, ok := gasMeter.(*fixedGasMeter); ok {
		if gasMeter.GasConsumed() == ZeroTxGas && gasMeter.Limit() == math.MaxUint64 {
			return true
		}
	}
	return false
}

// checkTxFeeWithValidatorMinGasPrices implements the default fee logic, where
// the minimum price per unit of gas is fixed and set by each validator, can the
// tx priority is computed from the gas price.
func checkTxFeeWithValidatorMinGasPrices(
	ctx sdk.Context,
	tx sdk.Tx,
) (sdk.Coins, int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, 0, sdkioerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()

	// Ensure that the provided fees meet a minimum threshold for the validator,
	// if this is a CheckTx. This is only for local mempool purposes, and thus
	// is only ran on check tx.
	if ctx.IsCheckTx() {
		minGasPrices := ctx.MinGasPrices()
		if !minGasPrices.IsZero() {
			requiredFees := make(sdk.Coins, len(minGasPrices))

			// Determine the required fees by multiplying each required minimum gas
			// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
			glDec := sdkmath.LegacyNewDec(int64(gas))
			for i, gp := range minGasPrices {
				fee := gp.Amount.Mul(glDec)
				requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
			}

			if !feeCoins.IsAnyGTE(requiredFees) {
				return nil, 0, sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", feeCoins, requiredFees)
			}
		}
	}

	priority := getTxPriority(feeCoins, int64(gas))
	return feeCoins, priority, nil
}

// getTxPriority returns a naive tx priority based on the amount of the smallest
// denomination of the gas price provided in a transaction.
// NOTE: This implementation should be used with a great consideration as it
// opens potential attack vectors where txs with multiple coins could not be
// prioritize as expected.
func getTxPriority(fee sdk.Coins, gas int64) int64 {
	var priority int64
	for _, c := range fee {
		p := int64(math.MaxInt64)
		gasPrice := c.Amount.QuoRaw(gas)
		if gasPrice.IsInt64() {
			p = gasPrice.Int64()
		}
		if priority == 0 || p < priority {
			priority = p
		}
	}

	return priority
}
