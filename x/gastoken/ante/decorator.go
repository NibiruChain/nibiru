package ante

import (
	"fmt"
	"math"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/keeper"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
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
	gasTokenKeeper *keeper.Keeper

	txFeeChecker authante.TxFeeChecker
}

func NewDeductFeeDecorator(gtk *keeper.Keeper, ek *evmkeeper.Keeper, ak types.AccountKeeper, bk authtypes.BankKeeper, fk types.FeegrantKeeper, tfc authante.TxFeeChecker) DeductFeeDecorator {
	if tfc == nil {
		tfc = checkTxFeeWithValidatorMinGasPrices
	}
	return DeductFeeDecorator{
		ak:             ak,
		evmkeeper:      ek,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		gasTokenKeeper: gtk,
		txFeeChecker:   tfc,
	}
}

func (dfd DeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkioerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	// checks to make sure the module account has been set to collect fees in base token
	if addr := dfd.ak.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}

	// fees can be in any denom (checked for validity later)
	fees := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	if len(fees) > 1 {
		return ctx, sdkioerrors.Wrapf(sdkerrors.ErrInvalidRequest, "only one fee token is supported, got %d", len(fees))
	}

	// set the fee payer as the default address to deduct fees from
	deductFeesFrom := feePayer

	// If a fee granter was set, deduct fee from the fee granter's account.
	if feeGranter != nil {
		if dfd.feegrantKeeper == nil {
			return ctx, sdkioerrors.Wrap(sdkerrors.ErrInvalidRequest, "fee grants is not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := dfd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fees, tx.GetMsgs())
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

	if simulate && fees.IsZero() {
		fees = sdk.NewCoins(sdk.NewInt64Coin(appconst.BondDenom, 1))
		burnAcctAddr, _ := sdk.AccAddressFromBech32("nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl")
		// were doing 1 extra get account call alas
		burnAcct := dfd.ak.GetAccount(ctx, burnAcctAddr)
		if burnAcct != nil {
			deductFeesFromAcc = burnAcct
		}
	}

	var (
		priority int64
		err      error
	)

	if !simulate {
		fees, priority, err = dfd.txFeeChecker(ctx, tx)
		if err != nil {
			return ctx, err
		}
	}

	// deducts the fees and transfer them to the module account
	if !fees.IsZero() {
		err = DeductFees(dfd.ak, dfd.evmkeeper, dfd.gasTokenKeeper, dfd.bankKeeper, ctx, deductFeesFromAcc, fees)
		if err != nil {
			return ctx, err
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{sdk.NewEvent(sdk.EventTypeTx,
		sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
	)})

	newCtx := ctx.WithPriority(priority)

	return next(newCtx, tx, simulate)
}

func DeductFees(accountkeeper types.AccountKeeper, ek *evmkeeper.Keeper, gtk *keeper.Keeper, bankKeeper authtypes.BankKeeper, ctx sdk.Context, acc authtypes.AccountI, fees sdk.Coins) error {
	// Checks the validity of the fee tokens (sorted, have positive amount, valid and unique denomination)
	if !fees.IsValid() {
		return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	if fees[0].Denom == appconst.BondDenom {
		err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), authtypes.FeeCollectorName, fees)
		if err == nil {
			return nil
		}

		// First fall back to wnibi

		params, err := gtk.GetParams(ctx)
		if err != nil {
			return sdkioerrors.Wrapf(err, "failed to get gastoken params")
		}
		wnibi := params.WnibiAddress

		wnibiBal, err := ek.GetErc20Balance(ctx, eth.NibiruAddrToEthAddr(acc.GetAddress()), gethcommon.HexToAddress(wnibi))
		if err != nil {
			return sdkioerrors.Wrapf(err, "failed to get WNIBI balance for account %s", acc.GetAddress())
		}
		feeCollector := eth.NibiruAddrToEthAddr(accountkeeper.GetModuleAddress(types.ModuleName))
		sender := eth.NibiruAddrToEthAddr(acc.GetAddress())
		nonce := ek.GetAccNonce(ctx, sender)

		feesAmount := fees[0].Amount

		if wnibiBal.Cmp(evm.NativeToWei(feesAmount.BigInt())) >= 0 {
			// If the user has enough WNIBI, just deduct in WNIBI
			err = ek.Erc20Transfer(ctx, gethcommon.HexToAddress(wnibi), sender, feeCollector, evm.NativeToWei(feesAmount.BigInt()))
			if err != nil {
				return sdkioerrors.Wrapf(err, "failed to transfer WNIBI from %s to fee collector", acc.GetAddress())
			}

			if err := WithdrawFeeToken(ctx, ek, accountkeeper, gethcommon.HexToAddress(wnibi), feeCollector, big.NewInt(0)); err != nil {
				return sdkioerrors.Wrapf(err, "failed to withdraw base token %s", gethcommon.HexToAddress(wnibi))
			}
			return nil
		}

		gasTokens := gtk.GetFeeTokens(ctx)
		for _, gasToken := range gasTokens {
			erc20Addr := gethcommon.HexToAddress(gasToken.Erc20Address)
			bal, err := ek.GetErc20Balance(ctx, eth.NibiruAddrToEthAddr(acc.GetAddress()), erc20Addr)
			if err != nil {
				return sdkioerrors.Wrapf(err, "failed to get ERC20 balance for account %s", acc.GetAddress())
			}

			amountNeeded, err := GetAmountInFromUniswap(
				ctx,
				ek,
				gtk,
				erc20Addr,
				gethcommon.HexToAddress(wnibi),
				big.NewInt(3000),
				evm.NativeToWei(feesAmount.BigInt()),
			)
			if err != nil {
				return sdkioerrors.Wrapf(err, "failed to get amount in from uniswap for token %s", gasToken.Erc20Address)
			}
			if bal.Cmp(amountNeeded) >= 0 {
				// If the user has enough of this gas token, swap and pay
				if err := SwapFeeToken(ctx, ek, accountkeeper, gtk, eth.NibiruAddrToEthAddr(acc.GetAddress()), gethcommon.HexToAddress(gasToken.Erc20Address), feeCollector, amountNeeded, evm.NativeToWei(feesAmount.BigInt())); err != nil {
					return sdkioerrors.Wrapf(err, "failed to swap gas token %s", gasToken.Erc20Address)
				}

				if err := WithdrawFeeToken(ctx, ek, accountkeeper, gethcommon.HexToAddress(wnibi), feeCollector, big.NewInt(0)); err != nil {
					return sdkioerrors.Wrapf(err, "failed to withdraw base token %s", gethcommon.HexToAddress(wnibi))
				}

				if err := acc.SetSequence(nonce); err != nil {
					return sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce)
				}

				accountkeeper.SetAccount(ctx, acc)

				return nil
			}
		}

		return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient balance across supported gas tokens to cover %s", feesAmount)

	} else {
		return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFee, "fee denom must be %s, got %s", appconst.BondDenom, fees[0].Denom)
	}
}

func WithdrawFeeToken(ctx sdk.Context, ek *evmkeeper.Keeper, ak types.AccountKeeper, contract, feeCollector gethcommon.Address, unusedBigInt *big.Int) error {
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
		GasLimit:         evmkeeper.Erc20GasLimitExecute,
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
		return fmt.Errorf("WNIBI withdraw failed: %s", resp.VmError)
	}

	if err := stateDB.Commit(); err != nil {
		return sdkioerrors.Wrap(err, "failed to commit stateDB")
	}

	acc := ak.GetModuleAccount(ctx, types.ModuleName)
	if err := acc.SetSequence(nonce); err != nil {
		return sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce)
	}

	return nil
}

func SwapFeeToken(ctx sdk.Context, ek *evmkeeper.Keeper, ak types.AccountKeeper, gtk *keeper.Keeper, sender gethcommon.Address, feeToken gethcommon.Address, feeCollector gethcommon.Address, amountIn, amountOut *big.Int) error {
	txFeesParams, err := gtk.GetParams(ctx)
	if err != nil {
		return sdkioerrors.Wrapf(err, "failed to get tx fees params")
	}

	routerAddr := gethcommon.HexToAddress(txFeesParams.UniswapV3SwapRouterAddress)
	err = ek.Erc20Approve(ctx, feeToken, sender, routerAddr, amountIn)
	if err != nil {
		return sdkioerrors.Wrapf(err, "failed to approve UniswapV3Pool contract %s for fee token %s", routerAddr.Hex(), feeToken)
	}

	// Pack swap call data using Uniswap V3 ABI
	input, err := embeds.SmartContract_UniswapV3SwapRouter.ABI.Pack(
		"exactOutputSingle",
		struct {
			TokenIn           gethcommon.Address
			TokenOut          gethcommon.Address
			Fee               *big.Int
			Recipient         gethcommon.Address
			AmountOut         *big.Int
			AmountInMaximum   *big.Int
			SqrtPriceLimitX96 *big.Int
		}{
			TokenIn:           feeToken,                                           // tokenIn
			TokenOut:          gethcommon.HexToAddress(txFeesParams.WnibiAddress), // tokenOut (WNIBI)
			Fee:               big.NewInt(3000),                                   // 0.3% pool
			Recipient:         feeCollector,                                       // who gets the WNIBI
			AmountOut:         amountOut,                                          // exact WNIBI we want
			AmountInMaximum:   amountIn,                                           // max tokenIn allowed
			SqrtPriceLimitX96: big.NewInt(0),                                      // no price limit
		},
	)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to pack ABI args for swap")
	}

	nonce := ek.GetAccNonce(ctx, sender)
	evmMsg := core.Message{
		To:               &routerAddr,
		From:             sender,
		Nonce:            nonce,
		Value:            big.NewInt(0),
		GasLimit:         evmkeeper.Erc20GasLimitExecute,
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

	resp, err := ek.CallContractWithInput(ctx, evmObj, sender, &routerAddr, false, input, evmkeeper.GetCallGasWithLimit(ctx, evmkeeper.Erc20GasLimitExecute))
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to call UniswapV3Pool swap")
	}
	if resp.Failed() {
		return sdkioerrors.Wrap(err, "UniswapV3Pool swap VM error")
	}
	if err := stateDB.Commit(); err != nil {
		return sdkioerrors.Wrap(err, "failed to commit stateDB after swap")
	}

	return nil
}

// getAmountInFromUniswap quotes the input needed to receive `amountOut` of tokenOut,
// using Uniswap V3 QuoterV2 if available, with a safe fallback that decodes QuoterV1-style reverts.
func GetAmountInFromUniswap(
	ctx sdk.Context,
	ek *evmkeeper.Keeper,
	gtk *keeper.Keeper,
	tokenIn gethcommon.Address,
	tokenOut gethcommon.Address,
	fee *big.Int,
	amountOut *big.Int,
) (*big.Int, error) {
	if amountOut == nil || amountOut.Sign() <= 0 {
		return nil, fmt.Errorf("amountOut must be > 0")
	}

	// --- Load addresses from params
	txFeesParams, err := gtk.GetParams(ctx)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "get tx fees params")
	}

	// Prefer QuoterV2 if configured; else try legacy Quoter
	var quoterAddr gethcommon.Address
	if txFeesParams.UniswapV3QuoterAddress != "" {
		quoterAddr = gethcommon.HexToAddress(txFeesParams.UniswapV3QuoterAddress)
	} else {
		return nil, fmt.Errorf("no Quoter address configured")
	}

	// --- Try QuoterV2 first (no revert; returns a tuple)
	type quoteExactOutputSingleParams struct {
		TokenIn           gethcommon.Address
		TokenOut          gethcommon.Address
		Amount            *big.Int // amountOut
		Fee               *big.Int
		SqrtPriceLimitX96 *big.Int // 0 = no limit
	}

	// Build the calldata for QuoterV2.quoteExactOutputSingle(params)
	var input []byte
	input, err = embeds.SmartContract_UniswapV3Quoter.ABI.Pack(
		"quoteExactOutputSingle",
		quoteExactOutputSingleParams{
			TokenIn:           tokenIn,
			TokenOut:          tokenOut,
			Amount:            amountOut,
			Fee:               fee,
			SqrtPriceLimitX96: big.NewInt(0),
		},
	)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "pack QuoterV2.quoteExactOutputSingle")
	}

	// --- Build a read-only EVM call (no state changes)
	evmMsg := core.Message{
		To:               &quoterAddr,
		From:             gethcommon.Address{}, // arbitrary
		Nonce:            0,
		Value:            big.NewInt(0),
		GasLimit:         2_500_000,
		GasPrice:         big.NewInt(0),
		GasFeeCap:        big.NewInt(0),
		GasTipCap:        big.NewInt(0),
		Data:             input,
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  true,
		SkipFromEOACheck: true,
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

	resp, err := ek.CallContractWithInput(ctx, evmObj, gethcommon.Address{}, &quoterAddr, false, input, evmkeeper.GetCallGasWithLimit(ctx, evmkeeper.Erc20GasLimitExecute))
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "quoter eth_call failed")
	}

	// --- Decode result
	// QuoterV2: success (no revert), returns tuple:
	// (uint256 amountIn, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate)
	if resp.Failed() {
		return nil, fmt.Errorf("QuoterV2 call failed: %s", resp.VmError)
	}
	// Unpack into struct to read amountIn
	var out struct {
		AmountIn                *big.Int
		SqrtPriceX96After       *big.Int
		InitializedTicksCrossed uint32
		GasEstimate             *big.Int
	}
	if err := embeds.SmartContract_UniswapV3Quoter.ABI.UnpackIntoInterface(&out, "quoteExactOutputSingle", resp.Ret); err != nil {
		return nil, sdkioerrors.Wrap(err, "unpack QuoterV2.quoteExactOutputSingle")
	}
	if out.AmountIn == nil || out.AmountIn.Sign() < 0 {
		return nil, fmt.Errorf("invalid amountIn from QuoterV2")
	}
	return out.AmountIn, nil
}

// checkTxFeeWithValidatorMinGasPrices implements the default fee logic, where the minimum price per
// unit of gas is fixed and set by each validator, can the tx priority is computed from the gas price.
func checkTxFeeWithValidatorMinGasPrices(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, 0, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
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
				return nil, 0, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", feeCoins, requiredFees)
			}
		}
	}

	priority := getTxPriority(feeCoins, int64(gas))
	return feeCoins, priority, nil
}

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
