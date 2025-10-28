package ante_test

import (
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func unibi(x int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, x)) }

// TestDeductFeeDecorator_ZeroGas verifies the behavior of the custom DeductFeeDecorator
// when a transaction declares zero gas:
//   - Setup: create a simple wasm MsgExecuteContract tx with gas limit set to 0 and
//     a funded sender account (so account lookup and signing succeed).
//   - Exercise: run the ante chain with IsCheckTx=true both in normal mode
//     (simulate=false) and in simulation mode (simulate=true).
//   - Expectation: in normal CheckTx, the decorator should reject the tx with an
//     invalid gas error; in simulation mode, the tx should be accepted to allow
//     client-side estimation flows.
func (s *Suite) TestDeductFeeDecorator_ZeroGas() {
	deps := evmtest.NewTestDeps()
	sender := deps.Sender
	s.Require().NoError(testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), sender.NibiruAddr, unibi(1)))

	// Build a simple tx with gas limit 0
	txCfg := deps.App.GetTxConfig()
	txBuilder := txCfg.NewTxBuilder()
	msg := &wasm.MsgExecuteContract{ // any valid msg to satisfy validation path
		Sender:   sender.NibiruAddr.String(),
		Contract: sender.NibiruAddr.String(), // not used downstream; ante doesn't decode
		Msg:      wasm.RawContractMessage([]byte("{}")),
		Funds:    sdk.Coins{},
	}
	s.Require().NoError(txBuilder.SetMsgs(msg))
	txBuilder.SetGasLimit(0)
	// no fees

	acc := deps.App.AccountKeeper.GetAccount(deps.Ctx(), sender.NibiruAddr)
	s.Require().NotNil(acc)
	// Create signed tx
	blockTx, err := s.CreateTestTx(
		txBuilder,
		[]cryptotypes.PrivKey{sender.PrivKey},
		[]uint64{acc.GetAccountNumber()},
		[]uint64{acc.GetSequence()},
		deps.Ctx().ChainID(),
		txCfg,
	)
	s.Require().NoError(err)

	dfd := ante.NewDeductFeeDecorator(deps.App.AccountKeeper, deps.App.BankKeeper, deps.App.FeeGrantKeeper, nil)
	antehandler := sdk.ChainAnteDecorators(dfd)

	// Set IsCheckTx to true and expect error on zero gas (simulate=false)
	ctx := deps.Ctx().WithIsCheckTx(true)
	_, err = antehandler(ctx, blockTx, false)
	s.Require().Error(err)

	// zero gas is accepted in simulation mode
	_, err = antehandler(ctx, blockTx, true)
	s.Require().NoError(err)
}

// TestEnsureMempoolFees validates fee checks against validator min gas prices and
// tx priority calculation in the mempool path:
// - Setup: build a signed tx with explicit gas limit and fee amount.
// - Exercise:
//  1. Set a high min gas price in CheckTx so the fee is insufficient -> expect error.
//  2. Run the same with simulate=true -> expect success (bypass check in simulation).
//  3. Run with IsCheckTx=false (DeliverTx) -> expect success (mempool min gas check disabled).
//  4. Set a very low min gas price in CheckTx so the fee is sufficient -> expect success,
//     and the returned context priority equals the smallest gas price amount (10 here).
func (s *Suite) TestEnsureMempoolFees() {
	deps := evmtest.NewTestDeps()
	sender := deps.Sender
	s.Require().NoError(testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), sender.NibiruAddr, unibi(1_000_000)))

	txCfg := deps.App.GetTxConfig()
	txBuilder := txCfg.NewTxBuilder()
	msg := &wasm.MsgExecuteContract{
		Sender:   sender.NibiruAddr.String(),
		Contract: sender.NibiruAddr.String(),
		Msg:      wasm.RawContractMessage([]byte("{}")),
	}
	s.Require().NoError(txBuilder.SetMsgs(msg))
	feeAmount := unibi(150) // some fee
	gasLimit := uint64(15)
	txBuilder.SetFeeAmount(feeAmount)
	txBuilder.SetGasLimit(gasLimit)

	acc := deps.App.AccountKeeper.GetAccount(deps.Ctx(), sender.NibiruAddr)
	blockTx, err := s.CreateTestTx(
		txBuilder, []cryptotypes.PrivKey{sender.PrivKey},
		[]uint64{acc.GetAccountNumber()},
		[]uint64{acc.GetSequence()},
		deps.Ctx().ChainID(), txCfg,
	)
	s.Require().NoError(err)

	dfd := ante.NewDeductFeeDecorator(deps.App.AccountKeeper, deps.App.BankKeeper, deps.App.FeeGrantKeeper, nil)
	antehandler := sdk.ChainAnteDecorators(dfd)

	// Set high gas price so standard test fee fails
	atomPrice := sdk.NewDecCoinFromDec("unibi", sdk.NewDec(20))
	highGasPrice := []sdk.DecCoin{atomPrice}
	ctx := deps.Ctx().WithMinGasPrices(highGasPrice).WithIsCheckTx(true)

	// should error with insufficient fees
	_, err = antehandler(ctx, blockTx, false)
	s.Require().Error(err)

	// simulation mode should pass
	cacheCtx, _ := ctx.CacheContext()
	_, err = antehandler(cacheCtx, blockTx, true)
	s.Require().NoError(err)

	// DeliverTx path (IsCheckTx=false) should not check min gas price
	ctx = ctx.WithIsCheckTx(false)
	_, err = antehandler(ctx, blockTx, false)
	s.Require().NoError(err)

	// back to CheckTx with low min gas price; should pass and set priority = smallest gas price amount
	ctx = ctx.WithIsCheckTx(true)
	atomPrice = sdk.NewDecCoinFromDec("unibi", sdk.NewDec(0).Quo(sdk.NewDec(100000)))
	lowGasPrice := []sdk.DecCoin{atomPrice}
	ctx = ctx.WithMinGasPrices(lowGasPrice)

	newCtx, err := antehandler(ctx, blockTx, false)
	s.Require().NoError(err)
	s.Require().Equal(int64(10), newCtx.Priority())
}

// TestDeductFees ensures the decorator deducts fees from the payer and returns
// an insufficient funds error when the account cannot cover the fee:
//   - Setup: build a signed tx with a non-zero fee and gas limit. Do not fund the
//     sender initially, so the first attempt fails.
//   - Exercise: run ante once to observe the insufficient funds error; then fund
//     the account and run again.
//   - Expectation: the first run errors; the second run succeeds after funding.
func (s *Suite) TestDeductFees() {
	deps := evmtest.NewTestDeps()
	sender := deps.Sender

	txCfg := deps.App.GetTxConfig()
	txBuilder := txCfg.NewTxBuilder()
	msg := &wasm.MsgExecuteContract{
		Sender:   sender.NibiruAddr.String(),
		Contract: sender.NibiruAddr.String(),
		Msg:      wasm.RawContractMessage([]byte("{}")),
	}
	s.Require().NoError(txBuilder.SetMsgs(msg))
	feeAmount := unibi(150)
	gasLimit := uint64(20000)
	txBuilder.SetFeeAmount(feeAmount)
	txBuilder.SetGasLimit(gasLimit)

	acc := deps.App.AccountKeeper.GetAccount(deps.Ctx(), sender.NibiruAddr)
	if acc == nil {
		// ensure account exists but without funds
		_ = testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), sender.NibiruAddr, unibi(0))
		acc = deps.App.AccountKeeper.GetAccount(deps.Ctx(), sender.NibiruAddr)
	}
	blockTx, err := s.CreateTestTx(
		txBuilder,
		[]cryptotypes.PrivKey{sender.PrivKey},
		[]uint64{acc.GetAccountNumber()},
		[]uint64{acc.GetSequence()},
		deps.Ctx().ChainID(),
		txCfg,
	)
	s.Require().NoError(err)

	dfd := ante.NewDeductFeeDecorator(deps.App.AccountKeeper, deps.App.BankKeeper, nil, nil)
	antehandler := sdk.ChainAnteDecorators(dfd)

	// Without funds, should error
	_, err = antehandler(deps.Ctx(), blockTx, false)
	s.Require().Error(err)

	// Fund account sufficiently and try again
	s.Require().NoError(testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), sender.NibiruAddr, unibi(1_000_000)))
	_, err = antehandler(deps.Ctx(), blockTx, false)
	s.Require().NoError(err)
}

// Additional test: when zero gas actor calls tx, zero gas meter set and fee is
// not deducted TestZeroGasActorSkipsFeeDeduction validates that when
// AnteDecZeroGasActors marks a tx as zero-gas (by setting a fixed zero
// GasMeter), the downstream custom DeductFeeDecorator skips fee deduction
// entirely.
//   - Setup: configure SudoKeeper.ZeroGasActors with a (sender, contract) pair,
//     fund the sender, and build a wasm execute tx targeting the whitelisted
//     contract with a non-zero fee amount to make fee changes observable.
//   - Exercise: run an ante chain that includes AnteDecZeroGasActors followed by
//     the [ante.DeductFeeDecorator].
//   - Expectation: the bank balance is unchanged before/after ante execution,
//     proving that no fees were deducted for zero-gas actors.
func (s *Suite) TestZeroGasActorSkipsFeeDeduction() {
	deps := evmtest.NewTestDeps()
	zeroGasSender := evmtest.NewEthPrivAcc()
	// Set zero gas actors to include zeroGasSender and a dummy contract
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	contract := wasmContracts[1]
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders:   []string{zeroGasSender.NibiruAddr.String()},
		Contracts: []string{contract.String()},
	})

	// Fund account with some balance
	s.Require().NoError(testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), zeroGasSender.NibiruAddr, unibi(10_000)))

	// Build tx that matches zero gas actors and also sets a non-zero fee to prove it's not deducted
	txCfg := deps.App.GetTxConfig()
	txBuilder := txCfg.NewTxBuilder()
	msg := &wasm.MsgExecuteContract{
		Sender:   zeroGasSender.NibiruAddr.String(),
		Contract: contract.String(),
		Msg:      wasm.RawContractMessage([]byte("{\n\t\"increment\": {}\n}")),
		Funds:    sdk.Coins{},
	}
	s.Require().NoError(txBuilder.SetMsgs(msg))
	txBuilder.SetGasLimit(100000)
	txBuilder.SetFeeAmount(unibi(1234))

	acc := deps.App.AccountKeeper.GetAccount(deps.Ctx(), zeroGasSender.NibiruAddr)
	blockTx, err := s.CreateTestTx(
		txBuilder,
		[]cryptotypes.PrivKey{zeroGasSender.PrivKey},
		[]uint64{acc.GetAccountNumber()},
		[]uint64{acc.GetSequence()},
		deps.Ctx().ChainID(),
		txCfg,
	)
	s.Require().NoError(err)

	// Chain AnteDecZeroGasActors then DeductFeeDecorator
	antehandler := sdk.ChainAnteDecorators(
		ante.AnteDecZeroGasActors{PublicKeepers: deps.App.PublicKeepers},
		ante.NewDeductFeeDecorator(deps.App.AccountKeeper, deps.App.BankKeeper, deps.App.FeeGrantKeeper, nil),
	)

	balBefore := deps.App.BankKeeper.GetBalance(deps.Ctx(), zeroGasSender.NibiruAddr, appconst.DENOM_UNIBI)

	newCtx, err := antehandler(deps.Ctx(), blockTx, false)
	s.Require().NoError(err)
	_ = newCtx

	balAfter := deps.App.BankKeeper.GetBalance(deps.Ctx(), zeroGasSender.NibiruAddr, appconst.DENOM_UNIBI)
	s.Require().Equal(balBefore, balAfter, "expect zero fees deducted for zero gas actor")
}
