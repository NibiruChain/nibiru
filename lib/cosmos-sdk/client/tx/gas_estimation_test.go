package tx

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keyring"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/multisig"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	txtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx/signing"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

type simulationRecorder struct {
	mockContext
	t      *testing.T
	config client.TxConfig
	calls  []sdk.FeeTx
	gas    func(sdk.FeeTx, int) (*sdk.GasInfo, error)
}

func (m *simulationRecorder) Invoke(_ context.Context, method string, req, reply interface{}, _ ...grpc.CallOption) error {
	require.Equal(m.t, "/cosmos.tx.v1beta1.Service/Simulate", method)
	decoded, err := m.config.TxDecoder()(req.(*txtypes.SimulateRequest).TxBytes)
	require.NoError(m.t, err)
	tx := decoded.(sdk.FeeTx)
	m.calls = append(m.calls, tx)
	info, err := m.gas(tx, len(m.calls))
	if err != nil {
		return err
	}
	reply.(*txtypes.SimulateResponse).GasInfo = info
	return nil
}

func TestCalculateGasFeeBearingCandidates(t *testing.T) {
	for _, adjustment := range []float64{1, 1.05} {
		t.Run(fmt.Sprint(adjustment), func(t *testing.T) {
			config, _ := newTestTxConfig(t)
			f := Factory{}.WithTxConfig(config).WithChainID("test").WithGasPrices("0.025unibi").WithGasAdjustment(adjustment)
			recorder := &simulationRecorder{t: t, config: config, gas: func(tx sdk.FeeTx, _ int) (*sdk.GasInfo, error) {
				gas := uint64(36722)
				if !tx.GetFee().IsZero() {
					gas = 53751
				}
				return &sdk.GasInfo{GasUsed: gas}, nil
			}}
			res, limit, err := CalculateGas(recorder, f)
			require.NoError(t, err)
			require.EqualValues(t, 53751, res.GasInfo.GasUsed)
			require.GreaterOrEqual(t, limit, uint64(53751+3429))
			require.Len(t, recorder.calls, 3)
			require.Zero(t, recorder.calls[0].GetGas())
			require.True(t, recorder.calls[0].GetFee().IsZero())
			for _, call := range recorder.calls[1:] {
				require.Positive(t, call.GetGas())
				require.False(t, call.GetFee().IsZero())
				sigs, err := call.(interface {
					GetSignaturesV2() ([]signing.SignatureV2, error)
				}).GetSignaturesV2()
				require.NoError(t, err)
				require.Equal(t, config.SignModeHandler().DefaultMode(), sigs[0].Data.(*signing.SingleSignatureData).SignMode)
				require.Empty(t, sigs[0].Data.(*signing.SingleSignatureData).Signature)
			}
			require.Equal(t, limit, recorder.calls[2].GetGas())
			expectedFee := f.WithGas(limit)
			built, err := expectedFee.BuildUnsignedTx()
			require.NoError(t, err)
			require.Equal(t, built.GetTx().GetFee(), recorder.calls[2].GetFee())
		})
	}
}

func TestCalculateGasFailuresAndConvergence(t *testing.T) {
	cases := []struct {
		name    string
		gas     func(sdk.FeeTx, int) (*sdk.GasInfo, error)
		calls   int
		wantErr string
	}{
		{"bootstrap error", func(sdk.FeeTx, int) (*sdk.GasInfo, error) { return nil, fmt.Errorf("unavailable") }, 1, "unavailable"},
		{"followup error", func(_ sdk.FeeTx, n int) (*sdk.GasInfo, error) {
			if n == 2 {
				return nil, fmt.Errorf("insufficient funds")
			}
			return &sdk.GasInfo{GasUsed: 100}, nil
		}, 2, "insufficient funds"},
		{"missing gas info", func(sdk.FeeTx, int) (*sdk.GasInfo, error) { return nil, nil }, 1, "no gas information"},
		{"nonconvergence", func(tx sdk.FeeTx, _ int) (*sdk.GasInfo, error) { return &sdk.GasInfo{GasUsed: tx.GetGas() + 1}, nil }, 4, "did not stabilize"},
		{"fourth call verifies increase", func(_ sdk.FeeTx, n int) (*sdk.GasInfo, error) {
			gas := uint64(100)
			if n >= 3 {
				gas = 101
			}
			return &sdk.GasInfo{GasUsed: gas}, nil
		}, 4, ""},
		{"zero gas", func(sdk.FeeTx, int) (*sdk.GasInfo, error) {
			return &sdk.GasInfo{GasUsed: 0, GasWanted: math.MaxUint64}, nil
		}, 3, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config, _ := newTestTxConfig(t)
			recorder := &simulationRecorder{t: t, config: config, gas: tc.gas}
			res, limit, err := CalculateGas(recorder, Factory{}.WithTxConfig(config).WithChainID("test").WithGasAdjustment(1))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, limit, recorder.calls[len(recorder.calls)-1].GetGas())
				if tc.name == "zero gas" {
					require.Zero(t, limit)
					require.Zero(t, estimationAllowance(res))
				}
			}
			require.Len(t, recorder.calls, tc.calls)
		})
	}
}

func TestAdjustedGas(t *testing.T) {
	for _, tc := range []struct {
		used, allowance uint64
		adjustment      float64
		want            uint64
	}{
		{1, 0, 1.05, 2}, {100, 0, 1.05, 105}, {53691, 3429, 1, 57120},
		{1<<53 + 1, 0, 1, 1<<53 + 1}, {math.MaxUint64, 0, 1, math.MaxUint64},
	} {
		got, err := adjustedGas(tc.used, tc.allowance, tc.adjustment)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
	for _, adjustment := range []float64{0, -1, 0.99, math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := adjustedGas(0, 0, adjustment)
		require.Error(t, err)
	}
	_, err := adjustedGas(math.MaxUint64, 1, 1)
	require.Error(t, err)
	_, err = adjustedGas(math.MaxUint64, 0, 1.05)
	require.Error(t, err)
}

func TestFactoryCLIFeeDefaults(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		price, fee string
		gas        uint64
	}{
		{"omitted", nil, "0.025unibi", "", flags.DefaultGasLimit},
		{"explicit fees", []string{"--fees=123unibi"}, "", "123unibi", flags.DefaultGasLimit},
		{"explicit empty fees", []string{"--fees="}, "", "", flags.DefaultGasLimit},
		{"zero prices", []string{"--gas-prices=0unibi"}, "0.000000000000000000unibi", "", flags.DefaultGasLimit},
		{"empty prices", []string{"--gas-prices="}, "", "", flags.DefaultGasLimit},
		{"custom prices", []string{"--gas-prices=0.1unibi"}, "0.100000000000000000unibi", "", flags.DefaultGasLimit},
		{"manual gas", []string{"--gas=200000"}, "0.025unibi", "", 200000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			flags.AddTxFlagsToCmd(cmd)
			require.NoError(t, cmd.Flags().Parse(tc.args))
			config, _ := newTestTxConfig(t)
			f, err := NewFactoryCLI(client.Context{}.WithTxConfig(config).WithChainID("test"), cmd.Flags())
			require.NoError(t, err)
			price, err := sdk.ParseDecCoins(tc.price)
			require.NoError(t, err)
			require.Equal(t, price, f.GasPrices())
			require.Equal(t, tc.fee, f.Fees().String())
			require.Equal(t, tc.gas, f.Gas())
			require.Equal(t, 1.05, f.GasAdjustment())
			if tc.name == "manual gas" {
				require.False(t, f.SimulateAndExecute())
			}
			_, err = f.BuildUnsignedTx()
			require.NoError(t, err)
		})
	}
}

func TestCalculateGasExplicitFees(t *testing.T) {
	config, _ := newTestTxConfig(t)
	recorder := &simulationRecorder{t: t, config: config, gas: func(tx sdk.FeeTx, _ int) (*sdk.GasInfo, error) {
		require.Equal(t, "1378unibi", tx.GetFee().String())
		return &sdk.GasInfo{GasUsed: 53691}, nil
	}}
	f := Factory{}.WithTxConfig(config).WithChainID("test").WithFees("1378unibi").WithGasAdjustment(1)
	_, limit, err := CalculateGas(recorder, f)
	require.NoError(t, err)
	require.EqualValues(t, 59990, limit)
}

func TestCalculateGasIgnoresManualGasForDryRun(t *testing.T) {
	config, _ := newTestTxConfig(t)
	recorder := &simulationRecorder{t: t, config: config, gas: func(sdk.FeeTx, int) (*sdk.GasInfo, error) { return &sdk.GasInfo{GasUsed: 100}, nil }}
	f := Factory{}.WithTxConfig(config).WithChainID("test").WithGas(200000).WithGasAdjustment(1)
	_, limit, err := CalculateGas(recorder, f)
	require.NoError(t, err)
	require.Zero(t, recorder.calls[0].GetGas())
	require.EqualValues(t, 3529, limit)
}

func TestCalculateGasOracleFixedGas(t *testing.T) {
	config, cdc := newTestTxConfig(t)
	oracletypes.RegisterInterfaces(cdc.(*codec.ProtoCodec).InterfaceRegistry())
	for _, msgs := range [][]sdk.Msg{
		{&oracletypes.MsgAggregateExchangeRatePrevote{}},
		{&oracletypes.MsgAggregateExchangeRateVote{}},
		{&oracletypes.MsgAggregateExchangeRatePrevote{}, &oracletypes.MsgAggregateExchangeRateVote{}},
	} {
		recorder := &simulationRecorder{t: t, config: config, gas: func(sdk.FeeTx, int) (*sdk.GasInfo, error) {
			return &sdk.GasInfo{GasUsed: 500, GasWanted: math.MaxUint64}, nil
		}}
		f := Factory{}.WithTxConfig(config).WithChainID("test").WithGasAdjustment(1.05)
		res, limit, err := CalculateGas(recorder, f, msgs...)
		require.NoError(t, err)
		require.EqualValues(t, 525, limit)
		require.Zero(t, estimationAllowance(res, msgs...))
	}
	// An unconfined simulation meter alone does not identify a fixed charge.
	require.EqualValues(t, 3429, estimationAllowance(&txtypes.SimulateResponse{GasInfo: &sdk.GasInfo{GasUsed: 500, GasWanted: math.MaxUint64}}))
}

func TestCalculateGasMultisig(t *testing.T) {
	config, cdc := newTestTxConfig(t)
	kb := keyring.NewInMemory(cdc)
	pubkey := multisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{secp256k1.GenPrivKey().PubKey(), secp256k1.GenPrivKey().PubKey()})
	_, err := kb.SaveMultisig("multi", pubkey)
	require.NoError(t, err)
	f := Factory{}.WithTxConfig(config).WithKeybase(kb).WithFromName("multi").WithSimulateAndExecute(true).
		WithChainID("test").WithSignMode(signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON).WithGasPrices("0.025unibi").WithGasAdjustment(1)
	recorder := &simulationRecorder{t: t, config: config, gas: func(tx sdk.FeeTx, _ int) (*sdk.GasInfo, error) {
		sigs, err := tx.(interface {
			GetSignaturesV2() ([]signing.SignatureV2, error)
		}).GetSignaturesV2()
		require.NoError(t, err)
		data := sigs[0].Data.(*signing.MultiSignatureData)
		require.Len(t, data.Signatures, 2)
		for _, sig := range data.Signatures {
			require.Equal(t, signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON, sig.(*signing.SingleSignatureData).SignMode)
		}
		return &sdk.GasInfo{GasUsed: 100}, nil
	}}
	_, limit, err := CalculateGas(recorder, f)
	require.NoError(t, err)
	require.EqualValues(t, 6399, limit, "counter and fee collector allowances apply once per transaction")
}

func TestBuildUnsignedTxLargeGasFee(t *testing.T) {
	config, _ := newTestTxConfig(t)
	f := Factory{}.WithTxConfig(config).WithChainID("test").WithGas(math.MaxUint64).WithGasPrices("0.025unibi")
	built, err := f.BuildUnsignedTx()
	require.NoError(t, err)
	require.Equal(t, "461168601842738791unibi", built.GetTx().GetFee().String())
}

func TestFeeCollectorGasAllowance(t *testing.T) {
	for _, tc := range []struct {
		name string
		f    Factory
		want uint64
	}{
		{"no fees", Factory{}, 0},
		{"zero prices", Factory{}.WithGasPrices("0unibi"), 0},
		{"priced unibi", Factory{}.WithGasPrices("0.025unibi"), 2870},
		{"explicit unibi", Factory{}.WithFees("123unibi"), 2870},
		{"multiple denominations", Factory{}.WithFees("1uatom,2unibi"), 5740},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, feeCollectorGasAllowance(tc.f))
			zero := &txtypes.SimulateResponse{GasInfo: &sdk.GasInfo{GasUsed: 0}}
			require.Zero(t, totalEstimationAllowance(zero, tc.f))
			fixed := &txtypes.SimulateResponse{GasInfo: &sdk.GasInfo{GasUsed: 500}}
			require.Zero(t, totalEstimationAllowance(fixed, tc.f, &oracletypes.MsgAggregateExchangeRateVote{}))
		})
	}
}
