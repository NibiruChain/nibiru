package keeper

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/core"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

// DeductFeeDecorator deducts fees from the first signer of the tx.
// If the first signer does not have the funds to pay for the fees, we return an InsufficientFunds error.
// We call next AnteHandler if fees successfully deducted.
//
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type DeductFeeDecorator struct {
	ak             types.AccountKeeper
	evmkeeper      *evmkeeper.Keeper
	bankKeeper     authtypes.BankKeeper
	feegrantKeeper types.FeegrantKeeper
	txFeesKeeper   Keeper
}

func NewDeductFeeDecorator(tk Keeper, ek *evmkeeper.Keeper, ak types.AccountKeeper, bk authtypes.BankKeeper, fk types.FeegrantKeeper) DeductFeeDecorator {
	return DeductFeeDecorator{
		ak:             ak,
		evmkeeper:      ek,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		txFeesKeeper:   tk,
	}
}

func (dfd DeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkioerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	// checks to make sure the module account has been set to collect fees in base token
	if addr := dfd.ak.GetModuleAddress(types.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", types.FeeCollectorName)
	}

	// TODO: only 1 denom for fee and that denom is either accepted or base denom

	// fee can be in any denom (checked for validity later)
	fee := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	// set the fee payer as the default address to deduct fees from
	deductFeesFrom := feePayer

	// If a fee granter was set, deduct fee from the fee granter's account.
	if feeGranter != nil {
		if dfd.feegrantKeeper == nil {
			return ctx, sdkioerrors.Wrap(sdkerrors.ErrInvalidRequest, "fee grants is not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := dfd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, tx.GetMsgs())
			if err != nil {
				return ctx, sdkioerrors.Wrapf(err, "%s not allowed to pay fees from %s", feeGranter, feePayer)
			}
		}

		// if no errors, change the account that is charged for fees to the fee granter
		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := dfd.ak.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return ctx, sdkioerrors.Wrapf(sdkerrors.ErrUnknownAddress, "fee payer address: %s does not exist", deductFeesFrom)
	}

	fees := feeTx.GetFee()

	if simulate && fees.IsZero() {
		fees = sdk.NewCoins(sdk.NewInt64Coin("unibi", 1))
		burnAcctAddr, _ := sdk.AccAddressFromBech32("nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl")
		// were doing 1 extra get account call alas
		burnAcct := dfd.ak.GetAccount(ctx, burnAcctAddr)
		if burnAcct != nil {
			deductFeesFromAcc = burnAcct
		}
	}

	// deducts the fees and transfer them to the module account
	if !fees.IsZero() {
		err = DeductFees(dfd.ak, dfd.evmkeeper, dfd.txFeesKeeper, dfd.bankKeeper, ctx, deductFeesFromAcc, fees)
		if err != nil {
			return ctx, err
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{sdk.NewEvent(sdk.EventTypeTx,
		sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
	)})

	return next(ctx, tx, simulate)
}

func DeductFees(accountkeeper types.AccountKeeper, ek *evmkeeper.Keeper, txFeesKeeper types.TxFeesKeeper, bankKeeper authtypes.BankKeeper, ctx sdk.Context, acc authtypes.AccountI, fees sdk.Coins) error {
	// Checks the validity of the fee tokens (sorted, have positive amount, valid and unique denomination)
	if !fees.IsValid() {
		return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	// pulls base denom from TxFeesKeeper
	baseDenom, err := txFeesKeeper.GetBaseDenom(ctx)
	if err != nil {
		return err
	}

	feeTokens := txFeesKeeper.GetFeeTokens(ctx)

	if fees[0].Denom == baseDenom {
		err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
		if err != nil {
			return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}
	} else {
		// A random account
		// TODO: change to another account so that we can swap to Nibi in the future
		addr := gethcommon.HexToAddress("0x4675eAE0Cc880F0E0A0D130e6619Cef08012EE65")
		if err != nil {
			return fmt.Errorf("failed to parse address: %w", err)
		}

		amount := fees[0].Amount.BigInt()

		unusedBigInt := big.NewInt(0)
		to := gethcommon.HexToAddress(feeTokens[0].Denom)
		from := eth.NibiruAddrToEthAddr(acc.GetAddress())

		input, err := embeds.SmartContract_WNIBI.ABI.Pack(
			"transfer", addr, amount,
		)
		if err != nil {
			return sdkioerrors.Wrap(err, "failed to pack ABI args for transfer")
		}

		nonce := ek.GetAccNonce(ctx, from)
		evmMsg := core.Message{
			To:               &to,
			From:             from,
			Nonce:            nonce,
			Value:            unusedBigInt, // amount
			GasLimit:         5_500_000,
			GasPrice:         unusedBigInt,
			GasFeeCap:        unusedBigInt,
			GasTipCap:        unusedBigInt,
			Data:             input,
			AccessList:       gethcore.AccessList{},
			SkipNonceChecks:  false,
			SkipFromEOACheck: false,
		}

		txConfig := ek.TxConfig(ctx, gethcommon.Hash{})
		stateDB := ek.Bank.StateDB
		if stateDB == nil {
			stateDB = ek.NewStateDB(ctx, txConfig)
		}
		defer func() {
			ek.Bank.StateDB = nil
		}()
		evmObj := ek.NewEVM(ctx, evmMsg, ek.GetEVMConfig(ctx), nil /*tracer*/, stateDB)

		_, resp, err := ek.ERC20().Transfer(to, from, addr, amount, ctx, evmObj)
		if err != nil {
			return sdkioerrors.Wrap(err, "failed to call WNIBI contract transfer")
		}
		if resp.Failed() {
			return sdkioerrors.Wrap(err, "failed to call WNIBI contract transfer with VM error")
		}

		if err := stateDB.Commit(); err != nil {
			return sdkioerrors.Wrap(err, "failed to commit stateDB")
		}


		if err := acc.SetSequence(nonce); err != nil {
			return sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce)
		}

		accountkeeper.SetAccount(ctx, acc)
	}

	return nil
}
