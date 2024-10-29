// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
)

// TestVerifyFee asserts that the result of VerifyFee is the effective fee
// in units of micronibi per gas.
func (s *Suite) TestVerifyFee() {
	baseFeeMicronibi := evm.BASE_FEE_MICRONIBI

	type testCase struct {
		name             string
		txData           evm.TxData
		baseFeeMicronibi *big.Int
		wantCoinAmt      string
		wantErr          string
	}

	for _, getTestCase := range []func() testCase{
		func() testCase {
			txData := evmtest.ValidLegacyTx()
			effectiveFeeMicronibi := evm.WeiToNative(txData.EffectiveFeeWei(nil))
			return testCase{
				name:             "happy: legacy tx",
				txData:           txData,
				baseFeeMicronibi: baseFeeMicronibi,
				wantCoinAmt:      effectiveFeeMicronibi.String(),
				wantErr:          "",
			}
		},
		func() testCase {
			txData := evmtest.ValidLegacyTx()
			txData.GasLimit = gethparams.TxGas - 1
			effectiveFeeMicronibi := evm.WeiToNative(txData.EffectiveFeeWei(nil))
			return testCase{
				name:             "sad: gas limit lower than global tx gas cost",
				txData:           txData,
				baseFeeMicronibi: baseFeeMicronibi,
				wantCoinAmt:      effectiveFeeMicronibi.String(),
				wantErr:          "gas limit too low",
			}
		},
		func() testCase {
			txData := evmtest.ValidLegacyTx()

			// Set a gas price that would make the gas fee cap "too low", i.e.
			// lower than the base fee
			baseFeeWei := evm.NativeToWei(baseFeeMicronibi)
			lowGasPrice := sdkmath.NewIntFromBigInt(
				new(big.Int).Sub(baseFeeWei, big.NewInt(1)),
			)
			txData.GasPrice = &lowGasPrice

			effectiveFeeMicronibi := evm.WeiToNative(txData.EffectiveFeeWei(baseFeeWei))

			return testCase{
				name:             "happy: gas fee cap lower than base fee",
				txData:           txData,
				baseFeeMicronibi: baseFeeMicronibi,
				wantCoinAmt:      effectiveFeeMicronibi.String(),
				wantErr:          "",
			}
		},
		func() testCase {
			txData := evmtest.ValidLegacyTx()

			// Set the base fee per gas and user-configured fee per gas to 0.
			gasPrice := sdkmath.ZeroInt()
			txData.GasLimit = gethparams.TxGas // needed for intrinsic gas
			txData.GasPrice = &gasPrice
			baseFeeMicronibi := big.NewInt(0)

			// Expect a cost to be 0
			wantCoinAmt := "0"
			effectiveFeeMicronibi := evm.WeiToNative(txData.EffectiveFeeWei(nil))
			s.Require().Equal(wantCoinAmt, effectiveFeeMicronibi.String())

			return testCase{
				// This is impossible because base fee is 1 unibi, however this
				// case is technically valid.
				name:             "happy: the impossible zero case",
				txData:           txData,
				baseFeeMicronibi: baseFeeMicronibi,
				wantCoinAmt:      "0",
				wantErr:          "",
			}
		},
	} {
		feeDenom := evm.EVMBankDenom
		isCheckTx := true
		tc := getTestCase()
		s.Run(tc.name, func() {
			gotCoins, err := evmkeeper.VerifyFee(
				tc.txData, feeDenom, tc.baseFeeMicronibi, isCheckTx,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
			s.Equal(tc.wantCoinAmt, gotCoins.AmountOf(feeDenom).String())
		})
	}
}

// TestRefundGas: Verifies that `Keeper.RefundGas` refunds properly with
// different values of effective gas price (weiPerGas) and fee collector balances.
func (s *Suite) TestRefundGas() {
	type testCase struct {
		name        string
		msgFrom     gethcommon.Address
		leftoverGas uint64
		// Comes from EffectiveGasPriceWeiPerGas
		weiPerGas *big.Int
		// (Optional) Expected error message that occurs from RefundGas
		wantErr string
		// refund amount is leftoverGas * weiPerGas * micronibiPerWei
		wantRefundAmt *big.Int
	}

	feeCollectorInitialBalance := big.NewInt(40_000)
	fundFeeCollectorEvmBal := func(deps *evmtest.TestDeps, s *Suite, bal *big.Int) {
		err := testapp.FundModuleAccount(
			deps.App.BankKeeper, deps.Ctx, auth.FeeCollectorName,
			sdk.NewCoins(sdk.NewCoin(
				evm.EVMBankDenom, math.NewIntFromBigInt(bal),
			)),
		)
		s.Require().NoError(err)
	}

	for _, getTestCase := range []func(deps *evmtest.TestDeps) testCase{
		func(deps *evmtest.TestDeps) testCase {
			fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:          "happy: geth tx gas, base fee normal",
				msgFrom:       deps.Sender.EthAddr,
				leftoverGas:   gethparams.TxGas,
				weiPerGas:     evm.BASE_FEE_WEI,
				wantErr:       "",
				wantRefundAmt: new(big.Int).SetUint64(gethparams.TxGas),
			}
		},
		func(deps *evmtest.TestDeps) testCase {
			fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:          "happy: minimum wei per gas -> 0 refund",
				msgFrom:       deps.Sender.EthAddr,
				leftoverGas:   gethparams.TxGas,
				weiPerGas:     big.NewInt(1),
				wantErr:       "",
				wantRefundAmt: new(big.Int).SetUint64(0),
			}
		},
		func(deps *evmtest.TestDeps) testCase {
			fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:          "happy: wei per gas slightly below default base fee",
				msgFrom:       deps.Sender.EthAddr,
				leftoverGas:   gethparams.TxGas,
				weiPerGas:     new(big.Int).Sub(evm.BASE_FEE_WEI, big.NewInt(1)),
				wantErr:       "",
				wantRefundAmt: new(big.Int).SetUint64(20_999),
			}
		},
		func(deps *evmtest.TestDeps) testCase {
			fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:          "happy: wei per gas 10% of default base fee",
				msgFrom:       deps.Sender.EthAddr,
				leftoverGas:   gethparams.TxGas,
				weiPerGas:     new(big.Int).Quo(evm.BASE_FEE_WEI, big.NewInt(10)),
				wantErr:       "",
				wantRefundAmt: new(big.Int).SetUint64(2100),
			}
		},
		func(deps *evmtest.TestDeps) testCase {
			// fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:        "sad: geth tx gas, base fee normal, fee collector is broke",
				msgFrom:     deps.Sender.EthAddr,
				leftoverGas: gethparams.TxGas,
				weiPerGas:   evm.BASE_FEE_WEI,
				wantErr:     "failed to refund 21000 leftover gas (21000unibi)",
			}
		},
		func(deps *evmtest.TestDeps) testCase {
			fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:        "sad: geth tx gas, negative base fee (impossible but here for completeness",
				msgFrom:     deps.Sender.EthAddr,
				leftoverGas: gethparams.TxGas,
				weiPerGas:   new(big.Int).Neg(evm.BASE_FEE_WEI),
				wantErr:     evm.ErrInvalidRefund.Error(),
			}
		},
	} {
		deps := evmtest.NewTestDeps()
		tc := getTestCase(&deps)
		s.Run(tc.name, func() {
			fromBalBefore := deps.App.BankKeeper.GetBalance(
				deps.Ctx, deps.Sender.NibiruAddr, evm.EVMBankDenom,
			).Amount.BigInt()
			feeCollectorBalBefore := deps.App.BankKeeper.GetBalance(
				deps.Ctx,
				auth.NewModuleAddress(auth.FeeCollectorName),
				evm.EVMBankDenom,
			).Amount.BigInt()

			err := deps.EvmKeeper.RefundGas(
				deps.Ctx, tc.msgFrom, tc.leftoverGas, tc.weiPerGas,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			// refund amount is leftoverGas * weiPerGas * micronibiPerWei
			// msgFrom should have balance
			fromBalAfter := deps.App.BankKeeper.GetBalance(
				deps.Ctx, deps.Sender.NibiruAddr, evm.EVMBankDenom,
			).Amount.BigInt()
			feeCollectorBalAfter := deps.App.BankKeeper.GetBalance(
				deps.Ctx,
				auth.NewModuleAddress(auth.FeeCollectorName),
				evm.EVMBankDenom,
			).Amount.BigInt()

			s.Equal(
				new(big.Int).Sub(fromBalAfter, fromBalBefore).String(),
				tc.wantRefundAmt.String(),
				"sender balance did not get refunded as expected",
			)
			s.Equal(
				new(big.Int).Sub(feeCollectorBalAfter, feeCollectorBalBefore).String(),
				new(big.Int).Neg(tc.wantRefundAmt).String(),
				"fee collector did not refund as expected",
			)
		})
	}
}
