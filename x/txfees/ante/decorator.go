package ante

import (
	"fmt"
	"math/big"
	"strings"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	oraclekeeper "github.com/NibiruChain/nibiru/v2/x/oracle/keeper"
	txfeeskeeper "github.com/NibiruChain/nibiru/v2/x/txfees/keeper"
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
	txFeesKeeper   txfeeskeeper.Keeper
	oracleKeeper   oraclekeeper.Keeper
}

func NewDeductFeeDecorator(tk txfeeskeeper.Keeper, ek *evmkeeper.Keeper, ak types.AccountKeeper, bk authtypes.BankKeeper, fk types.FeegrantKeeper, ok oraclekeeper.Keeper) DeductFeeDecorator {
	return DeductFeeDecorator{
		ak:             ak,
		evmkeeper:      ek,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		txFeesKeeper:   tk,
		oracleKeeper:   ok,
	}
}

func (dfd DeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkioerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	// checks to make sure the module account has been set to collect fees in base token
	if addr := dfd.ak.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
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
		err = DeductFees(dfd.ak, dfd.evmkeeper, dfd.txFeesKeeper, dfd.bankKeeper, dfd.oracleKeeper, ctx, deductFeesFromAcc, fees)
		if err != nil {
			return ctx, err
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{sdk.NewEvent(sdk.EventTypeTx,
		sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
	)})

	return next(ctx, tx, simulate)
}

func DeductFees(accountkeeper types.AccountKeeper, ek *evmkeeper.Keeper, txFeesKeeper txfeeskeeper.Keeper, bankKeeper authtypes.BankKeeper, ok oraclekeeper.Keeper, ctx sdk.Context, acc authtypes.AccountI, fees sdk.Coins) error {
	// Checks the validity of the fee tokens (sorted, have positive amount, valid and unique denomination)
	if !fees.IsValid() {
		return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	if fees[0].Denom == appconst.BondDenom {
		err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), authtypes.FeeCollectorName, fees)
		if err != nil {
			return sdkioerrors.Wrap(sdkerrors.ErrInsufficientFunds, err.Error())
		}
	} else {
		feeToken, err := txFeesKeeper.GetFeeToken(ctx, strings.TrimPrefix(fees[0].Denom, "erc20/"))
		if err != nil {
			return sdkioerrors.Wrapf(sdkerrors.ErrInvalidRequest, "fee token %s not found, must follow the format erc20/{token_address}", fees[0].Denom)
		}

		var ratio sdkmath.LegacyDec
		if feeToken.TokenType == types.FeeTokenType_FEE_TOKEN_TYPE_SWAPPABLE {
			price, err := ok.GetExchangeRateTwap(ctx, asset.Pair(feeToken.Pair))
			if err != nil {
				return sdkioerrors.Wrapf(err, "failed to get exchange rate for pair %s", feeToken.Pair)
			}
			basePrice, err := ok.GetExchangeRateTwap(ctx, asset.Pair("unibi:uusd"))
			if err != nil {
				return sdkioerrors.Wrapf(err, "failed to get TWAP for unibi:uusd")
			}
			if price.IsZero() {
				return sdkioerrors.Wrapf(sdkerrors.ErrInvalidRequest, "price for %s is zero", feeToken.Pair)
			}
			ratio = basePrice.Quo(price)
		} else {
			ratio = sdkmath.LegacyOneDec()
		}
		feeCollector := eth.NibiruAddrToEthAddr(accountkeeper.GetModuleAddress(types.ModuleName))
		amount := sdkmath.LegacyNewDecFromInt(fees[0].Amount).Mul(ratio).TruncateInt().BigInt()
		sender := eth.NibiruAddrToEthAddr(acc.GetAddress())
		nonce := ek.GetAccNonce(ctx, sender)
		fmt.Println("amount :", amount.String(), "fee token:", feeToken.Address, "sender:", sender.Hex(), "fee collector:", feeCollector.Hex())
		err = ek.Erc20Transfer(ctx, gethcommon.HexToAddress(feeToken.Address), sender, feeCollector, amount)
		if err != nil {
			return err
		}

		if err := acc.SetSequence(nonce); err != nil {
			return sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce)
		}

		accountkeeper.SetAccount(ctx, acc)

		if feeToken.TokenType == types.FeeTokenType_FEE_TOKEN_TYPE_CONVERTIBLE {
			unusedBigInt := big.NewInt(0)
			err = withdrawFeeToken(ctx, ek, accountkeeper, gethcommon.HexToAddress(feeToken.Address), feeCollector, unusedBigInt)
			if err != nil {
				return sdkioerrors.Wrapf(err, "failed to withdraw fee token %s", feeToken.Address)
			}
		} else {
			err = swapFeeToken(ctx, ek, accountkeeper, txFeesKeeper, feeToken, feeCollector)
			if err != nil {
				return sdkioerrors.Wrapf(err, "failed to swap fee token %s", feeToken.Address)
			}

			unusedBigInt := big.NewInt(0)
			err = withdrawFeeToken(ctx, ek, accountkeeper, gethcommon.HexToAddress(feeToken.Address), feeCollector, unusedBigInt)
			if err != nil {
				return sdkioerrors.Wrapf(err, "failed to withdraw fee token %s", feeToken.Address)
			}
		}

	}

	return nil
}

func withdrawFeeToken(ctx sdk.Context, ek *evmkeeper.Keeper, ak types.AccountKeeper, contract, feeCollector gethcommon.Address, unusedBigInt *big.Int) error {
	out, err := ek.GetErc20Balance(ctx, feeCollector, contract)
	if err != nil {
		return fmt.Errorf("failed to get ERC20 balance: %w", err)
	}

	input, err := embeds.SmartContract_WNIBI.ABI.Pack(
		"withdraw", out,
	)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to pack ABI args for withdraw")
	}

	nonce := ek.GetAccNonce(ctx, feeCollector)
	evmMsg := core.Message{
		To:               &contract,
		From:             feeCollector,
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

	resp, err := ek.CallContractWithInput(ctx, evmObj, feeCollector, &contract, false /*commit*/, input, evmkeeper.GetCallGasWithLimit(ctx, evmkeeper.Erc20GasLimitExecute))
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to call WNIBI contract withdraw")
	}

	if resp.Failed() {
		return sdkioerrors.Wrap(err, "failed to call WNIBI contract withdraw with VM error")
	}

	if err := stateDB.Commit(); err != nil {
		return sdkioerrors.Wrap(err, "failed to commit stateDB")
	}

	acc := ak.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	if err := acc.SetSequence(nonce); err != nil {
		return sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce)
	}

	return nil
}

var (
	MinSqrtRatio    = new(big.Int).SetUint64(4295128739)
	MaxSqrtRatio, _ = new(big.Int).SetString("1461446703485210103287273052203988822378723970342", 10)
)

func swapFeeToken(ctx sdk.Context, ek *evmkeeper.Keeper, ak types.AccountKeeper, txfk txfeeskeeper.Keeper, feeToken types.FeeToken, feeCollector gethcommon.Address) error {
	poolAddr := gethcommon.HexToAddress(feeToken.PoolAddress)
	out, err := ek.GetErc20Balance(ctx, feeCollector, gethcommon.HexToAddress(feeToken.Address))
	if err != nil {
		return fmt.Errorf("failed to get ERC20 balance: %w", err)
	}

	err = ek.Erc20Approve(ctx, gethcommon.HexToAddress(feeToken.Address), feeCollector, poolAddr, out)
	if err != nil {
		return sdkioerrors.Wrapf(err, "failed to approve UniswapV3Pool contract %s for fee token %s", poolAddr.Hex(), feeToken.Address)
	}

	amountIn := out
	zeroForOne := true // tokenIn -> tokenOut
	var sqrtPriceLimitX96 *big.Int
	if zeroForOne {
		sqrtPriceLimitX96 = new(big.Int).Add(MinSqrtRatio, big.NewInt(1))
	} else {
		sqrtPriceLimitX96 = new(big.Int).Sub(MaxSqrtRatio, big.NewInt(1))
	}
	// // Pack swap call data using Uniswap V3 ABI
	// input, err := embeds.SmartContract_UniswapV3Pool.ABI.Pack(
	// 	"swap",
	// 	feeCollector,      // recipient
	// 	zeroForOne,        // zeroForOne
	// 	amountIn,          // amountSpecified
	// 	sqrtPriceLimitX96, // sqrtPriceLimitX96 (0 = no limit)
	// 	[]byte{},          // data (empty for direct swap)
	// )

	fmt.Println("sqrtPriceLimitX96 :", sqrtPriceLimitX96)
	baseToken, err := txfk.GetBaseToken(ctx, "WNIBI")

	if err != nil {
		return sdkioerrors.Wrapf(err, "failed to get base token for pair %s", feeToken.Pair)
	}
	if baseToken.Address == "" {
		return sdkioerrors.Wrapf(sdkerrors.ErrInvalidRequest, "base token address for pair %s is empty", feeToken.Pair)
	}
	// Pack swap call data using Uniswap V3 ABI
	input, err := embeds.SmartContract_UniswapV3SwapRouter.ABI.Pack(
		"exactInputSingle",
		struct {
			TokenIn           gethcommon.Address
			TokenOut          gethcommon.Address
			Fee               *big.Int
			Recipient         gethcommon.Address
			AmountIn          *big.Int
			AmountOutMinimum  *big.Int
			SqrtPriceLimitX96 *big.Int
		}{
			TokenIn:           gethcommon.HexToAddress(feeToken.Address),  // tokenIn
			TokenOut:          gethcommon.HexToAddress(baseToken.Address), // tokenOut
			Fee:               big.NewInt(3000),                           // fee
			Recipient:         feeCollector,                               // recipient                                // deadline
			AmountIn:          amountIn,                                   // amountSpecified
			AmountOutMinimum:  big.NewInt(0),                              // sqrtPriceLimitX96 (0 = no limit)
			SqrtPriceLimitX96: big.NewInt(0),                              // data (empty for direct swap)
		},
	)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to pack ABI args for swap")
	}

	nonce := ek.GetAccNonce(ctx, feeCollector)
	evmMsg := core.Message{
		To:               &poolAddr,
		From:             feeCollector,
		Nonce:            nonce,
		Value:            big.NewInt(0),
		GasLimit:         5_500_000,
		GasPrice:         big.NewInt(0),
		GasFeeCap:        big.NewInt(0),
		GasTipCap:        big.NewInt(0),
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
	evmObj := ek.NewEVM(ctx, evmMsg, ek.GetEVMConfig(ctx), nil, stateDB)

	resp, err := ek.CallContractWithInput(ctx, evmObj, feeCollector, &poolAddr, false, input, evmkeeper.GetCallGasWithLimit(ctx, evmkeeper.Erc20GasLimitExecute))
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to call UniswapV3Pool swap")
	}
	if resp.Failed() {
		return sdkioerrors.Wrap(err, "UniswapV3Pool swap VM error")
	}
	if err := stateDB.Commit(); err != nil {
		return sdkioerrors.Wrap(err, "failed to commit stateDB after swap")
	}

	acc := ak.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	if err := acc.SetSequence(nonce); err != nil {
		return sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce)
	}

	return nil
}
