// Copyright (c) 2023-2024 Nibi, Inc.
package evmstate_test

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"

	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

// TestVerifyFee asserts that the result of VerifyFee is the effective fee
// in units of micronibi per gas.
func (s *Suite) TestVerifyFee() {
	baseFeeMicronibi := evm.BASE_FEE_MICRONIBI

	type testCase struct {
		name             string
		txData           evm.TxData
		baseFeeMicronibi *big.Int
		wantWeiAmt       string
		wantErr          string
	}

	for _, getTestCase := range []func() testCase{
		func() testCase {
			txData := evmtest.ValidLegacyTx()
			effFeeWei := txData.EffectiveFeeWei(nil)
			return testCase{
				name:             "happy: legacy tx",
				txData:           txData,
				baseFeeMicronibi: baseFeeMicronibi,
				wantWeiAmt:       effFeeWei.String(),
				wantErr:          "",
			}
		},
		func() testCase {
			txData := evmtest.ValidLegacyTx()
			txData.GasLimit = gethparams.TxGas - 1
			effFeeWei := txData.EffectiveFeeWei(nil)
			return testCase{
				name:             "sad: gas limit lower than global tx gas cost",
				txData:           txData,
				baseFeeMicronibi: baseFeeMicronibi,
				wantWeiAmt:       effFeeWei.String(),
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

			effFeeWei := txData.EffectiveFeeWei(baseFeeWei)

			return testCase{
				name:             "happy: gas fee cap lower than base fee",
				txData:           txData,
				baseFeeMicronibi: baseFeeMicronibi,
				wantWeiAmt:       effFeeWei.String(),
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
			effFeeWei := txData.EffectiveFeeWei(nil)
			s.Require().Equal(wantCoinAmt, effFeeWei.String())

			return testCase{
				// This is impossible because base fee is 1 unibi, however this
				// case is technically valid.
				name:             "happy: the impossible zero case",
				txData:           txData,
				baseFeeMicronibi: baseFeeMicronibi,
				wantWeiAmt:       "0",
				wantErr:          "",
			}
		},
	} {
		tc := getTestCase()
		ctx := sdk.Context{}.WithIsCheckTx(true)
		s.Run(tc.name, func() {
			gotWeiFee, err := evmstate.VerifyFee(
				tc.txData, tc.baseFeeMicronibi, ctx,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
			s.Equal(tc.wantWeiAmt, gotWeiFee.String())
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
		wantRefundAmt string
	}

	feeCollectorInitialBalance := big.NewInt(40_000)
	fundFeeCollectorEvmBal := func(deps *evmtest.TestDeps, s *Suite, bal *big.Int) {
		err := testapp.FundModuleAccount(
			deps.App.BankKeeper, deps.Ctx(), auth.FeeCollectorName,
			sdk.NewCoins(sdk.NewCoin(
				evm.EVMBankDenom, sdkmath.NewIntFromBigInt(bal),
			)),
		)
		s.Require().NoError(err)
	}

	for _, getTestCase := range []func(deps *evmtest.TestDeps) testCase{
		func(deps *evmtest.TestDeps) testCase {
			fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:        "happy: geth tx gas, base fee normal",
				msgFrom:     deps.Sender.EthAddr,
				leftoverGas: gethparams.TxGas,
				weiPerGas:   evm.BASE_FEE_WEI,
				wantErr:     "",
				// wantRefundAmt = 21000 * 10^{12}
				wantRefundAmt: fmt.Sprintf("%d", gethparams.TxGas) + strings.Repeat("0", 12),
			}
		},
		func(deps *evmtest.TestDeps) testCase {
			fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:        "happy: wei per gas of 1 -> 0 refund",
				msgFrom:     deps.Sender.EthAddr,
				leftoverGas: gethparams.TxGas,
				weiPerGas:   big.NewInt(1),
				wantErr:     "",
				// wantRefundAmt = 21000 * 1
				wantRefundAmt: fmt.Sprintf("%d", gethparams.TxGas),
			}
		},
		func(deps *evmtest.TestDeps) testCase {
			fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:        "happy: wei per gas slightly below default base fee",
				msgFrom:     deps.Sender.EthAddr,
				leftoverGas: gethparams.TxGas,
				weiPerGas:   new(big.Int).Sub(evm.BASE_FEE_WEI, big.NewInt(1)),
				wantErr:     "",
				// wantRefundAmt = 21000 * (10^{12} - 1)
				wantRefundAmt: new(big.Int).Mul(
					big.NewInt(21_000),
					new(big.Int).Sub(
						new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil),
						big.NewInt(1),
					),
				).String(),
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
				wantRefundAmt: "2100" + strings.Repeat("0", 12),
			}
		},
		func(deps *evmtest.TestDeps) testCase {
			// fundFeeCollectorEvmBal(deps, s, feeCollectorInitialBalance)
			return testCase{
				name:        "sad: geth tx gas, base fee normal, fee collector is broke",
				msgFrom:     deps.Sender.EthAddr,
				leftoverGas: gethparams.TxGas,
				weiPerGas:   evm.BASE_FEE_WEI,
				wantErr:     "fee collector account failed to refund",
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
			sdb := deps.NewStateDB()

			fromBalBefore := sdb.GetBalance(deps.Sender.EthAddr)
			feeCollectorBalBefore := sdb.GetBalance(evm.FEE_COLLECTOR_ADDR)

			err := deps.EvmKeeper.RefundGas(
				deps.NewStateDB(), tc.msgFrom, tc.leftoverGas, tc.weiPerGas,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			// refund amount is leftoverGas * weiPerGas * micronibiPerWei
			// msgFrom should have balance
			fromBalAfter := sdb.GetBalance(deps.Sender.EthAddr)
			feeCollectorBalAfter := sdb.GetBalance(evm.FEE_COLLECTOR_ADDR)

			s.Equal(
				tc.wantRefundAmt,
				new(big.Int).Sub(fromBalAfter.ToBig(), fromBalBefore.ToBig()).String(),
				"sender balance did not get refunded as expected",
			)
			s.Equal(
				tc.wantRefundAmt,
				new(big.Int).Sub(feeCollectorBalBefore.ToBig(), feeCollectorBalAfter.ToBig()).String(),
				"fee collector did not refund as expected",
			)
		})
	}
}
