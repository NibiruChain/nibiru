package ante

import (
	"fmt"
	"math"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
)

var (
	_ sdk.AnteDecorator = DeductFeeDecorator{}
)

// DeductFeeDecorator deducts fees from the fee payer. The fee payer is the fee granter (if specified) or first signer of the tx.
// If the fee payer does not have the funds to pay for the fees, return an InsufficientFunds error.
// Call next AnteHandler if fees successfully deducted.
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type DeductFeeDecorator struct {
	accountKeeper  authkeeper.AccountKeeper
	evmkeeper      *evmkeeper.Keeper
	bankKeeper     types.BankKeeper
	feegrantKeeper FeegrantKeeper
	txFeeChecker   authante.TxFeeChecker
}

func NewDeductFeeDecorator(ak authkeeper.AccountKeeper, ek *evmkeeper.Keeper, bk types.BankKeeper, fk FeegrantKeeper, tfc authante.TxFeeChecker) DeductFeeDecorator {
	if tfc == nil {
		tfc = checkTxFeeWithValidatorMinGasPrices
	}

	return DeductFeeDecorator{
		accountKeeper:  ak,
		evmkeeper:      ek,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		txFeeChecker:   tfc,
	}
}

func (dfd DeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkioerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, sdkioerrors.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	pausedGasMeter := ctx.GasMeter()
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())

	var (
		priority int64
		err      error
	)

	fee := feeTx.GetFee()
	if !simulate {
		fee, priority, err = dfd.txFeeChecker(ctx, tx)
		if err != nil {
			return ctx, err
		}
	}
	if err := dfd.checkDeductFee(ctx, tx, fee); err != nil {
		return ctx, err
	}

	// TODO: print gas consumption values to verify this works as expected

	newCtx := ctx.WithPriority(priority).WithGasMeter(pausedGasMeter)
	fmt.Printf("newCtx.GasMeter().GasConsumed(): %v\n", newCtx.GasMeter().GasConsumed())

	return next(newCtx, tx, simulate)
}

func (dfd DeductFeeDecorator) checkDeductFee(ctx sdk.Context, sdkTx sdk.Tx, fee sdk.Coins) error {
	feeTx, ok := sdkTx.(sdk.FeeTx)
	if !ok {
		return sdkioerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if addr := dfd.accountKeeper.GetModuleAddress(types.FeeCollectorName); addr == nil {
		return fmt.Errorf("fee collector module account (%s) has not been set", types.FeeCollectorName)
	}

	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	deductFeesFrom := feePayer

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if dfd.feegrantKeeper == nil {
			return sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := dfd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, sdkTx.GetMsgs())
			if err != nil {
				return sdkioerrors.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := dfd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	// deduct the fees
	if !fee.IsZero() {
		err := DeductFees(dfd.accountKeeper, dfd.evmkeeper, dfd.bankKeeper, ctx, deductFeesFromAcc, fee)
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

// DeductFees deducts fees from the given account.
func DeductFees(accountKeeper authante.AccountKeeper, evmKeeper *evmkeeper.Keeper, bankKeeper types.BankKeeper, ctx sdk.Context, acc types.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	gasMeterBefore := ctx.GasMeter()
	gasConsumedBefore := gasMeterBefore.GasConsumed()
	baseOpGasConsumed := uint64(0)

	defer func() {
		// NOTE: we have to refund the entire gasMeterBefore because it's modified by AfterOp
		// stateDB.getStateObject() reads from state using the local root ctx which affects the gas meter
		gasMeterBefore.RefundGas(gasMeterBefore.GasConsumed(), "")
		gasMeterBefore.ConsumeGas(gasConsumedBefore+baseOpGasConsumed, "DeductFeeDecorator invariant")
	}()

	if fees[0].Denom == appconst.BondDenom {
		err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
		if err == nil {
			return nil
		}

		baseOpGasConsumed = ctx.GasMeter().GasConsumed()
		ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())

		// fallback to WNIBI
		err = DeductFeesWithWNIBI(ctx, accountKeeper, evmKeeper, acc, fees)
		if err == nil {
			return nil
		}

		return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient balance across supported gas tokens to cover %s", fees[0].Amount)
	} else {
		return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFee, "fee denom must be %s, got %s", appconst.BondDenom, fees[0].Denom)
	}
}

// DeductFeesWithWNIBI tries to deduct fees from WNIBI balance if native deduction fails.
func DeductFeesWithWNIBI(
	ctx sdk.Context,
	accountKeeper authante.AccountKeeper,
	evmKeeper *evmkeeper.Keeper,
	acc types.AccountI,
	fees sdk.Coins,
) error {
	wnibi := evmKeeper.GetParams(ctx).CanonicalWnibi

	stateDB := evmKeeper.Bank.StateDB
	if stateDB == nil {
		stateDB = evmKeeper.NewStateDB(ctx, evmKeeper.TxConfig(ctx, gethcommon.Hash{}))
	}
	defer func() {
		evmKeeper.Bank.StateDB = nil
	}()

	evmObj := evmKeeper.NewEVM(ctx, evm.MOCK_GETH_MESSAGE, evmKeeper.GetEVMConfig(ctx), nil, stateDB)
	wnibiBal, err := evmKeeper.ERC20().BalanceOf(wnibi.Address, eth.NibiruAddrToEthAddr(acc.GetAddress()), ctx, evmObj)
	if err != nil {
		return sdkioerrors.Wrapf(err, "failed to get WNIBI balance for account %s", acc.GetAddress())
	}

	feeCollector := eth.NibiruAddrToEthAddr(accountKeeper.GetModuleAddress(types.FeeCollectorName))
	feesAmount := fees[0].Amount

	sender := evm.Addrs{
		Bech32: acc.GetAddress(),
		Eth:    eth.NibiruAddrToEthAddr(acc.GetAddress()),
	}
	if wnibiBal.Cmp(evm.NativeToWei(feesAmount.BigInt())) >= 0 {
		nonce := evmKeeper.GetAccNonce(ctx, sender.Eth)
		_, err = evmKeeper.ConvertEvmToCoinForWNIBI(
			ctx, stateDB, wnibi, sender, accountKeeper.GetModuleAddress(types.FeeCollectorName),
			sdkmath.NewIntFromBigInt(evm.NativeToWei(feesAmount.BigInt())),
			nil,
		)
		if err != nil {
			return sdkioerrors.Wrapf(err, "failed to transfer WNIBI from %s to %s", sender.Eth.Hex(), feeCollector.Hex())
		}
		if err := acc.SetSequence(nonce); err != nil {
			return sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce)
		}
		accountKeeper.SetAccount(ctx, acc)
		return nil
	}
	return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient balance across supported gas tokens to cover %s", feesAmount)
}

// checkTxFeeWithValidatorMinGasPrices implements the default fee logic, where the minimum price per
// unit of gas is fixed and set by each validator, can the tx priority is computed from the gas price.
func checkTxFeeWithValidatorMinGasPrices(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
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

// getTxPriority returns a naive tx priority based on the amount of the smallest denomination of the gas price
// provided in a transaction.
// NOTE: This implementation should be used with a great consideration as it opens potential attack vectors
// where txs with multiple coins could not be prioritize as expected.
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
