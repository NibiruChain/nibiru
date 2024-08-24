// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"

	gethparams "github.com/ethereum/go-ethereum/params"

	sdkmath "cosmossdk.io/math"

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

			effectiveFeeMicronibi := evm.WeiToNative(txData.EffectiveFeeWei(nil))

			return testCase{
				name:             "sad: gas fee cap lower than base fee",
				txData:           txData,
				baseFeeMicronibi: baseFeeMicronibi,
				wantCoinAmt:      effectiveFeeMicronibi.String(),
				wantErr:          "gasfeecap is lower than the tx baseFee",
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
		feeDenom := evm.DefaultEVMDenom
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
