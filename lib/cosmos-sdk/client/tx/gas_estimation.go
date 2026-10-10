package tx

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"

	gogogrpc "github.com/cosmos/gogoproto/grpc"

	storetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/address"
	txtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
)

// TransactionCounterGasAllowance covers the read and write skipped by
// CountTXDecorator during simulation. The existing counter has a one-byte key
// and a 12-byte value. Charging for an existing value also covers an absent key.
func TransactionCounterGasAllowance() uint64 {
	config := storetypes.KVGasConfig()
	const counterBytes = 1 + 8 + 4
	return config.ReadCostFlat + config.ReadCostPerByte*counterBytes +
		config.WriteCostFlat + config.WriteCostPerByte*counterBytes
}

func estimationAllowance(res *txtypes.SimulateResponse, msgs ...sdk.Msg) uint64 {
	if res.GasInfo.GasUsed == 0 {
		return 0
	}
	// Oracle vote/prevote transactions reset the meter to a fixed charge after
	// the counter runs. Do not add ordinary SDK overhead to that charge.
	if len(msgs) > 0 {
		oracle := true
		for _, msg := range msgs {
			switch sdk.MsgTypeURL(msg) {
			case "/nibiru.oracle.v1.MsgAggregateExchangeRateVote", "/nibiru.oracle.v1.MsgAggregateExchangeRatePrevote":
			default:
				oracle = false
			}
		}
		if oracle {
			return 0
		}
	}
	return TransactionCounterGasAllowance()
}

// feeCollectorGasAllowance covers creating the denomination reverse index when
// BeginBlock drains the fee collector after the state used by simulation.
// Include one write per paid denomination, even if simulation already paid it.
func feeCollectorGasAllowance(f Factory) uint64 {
	denoms := make(map[string]bool)
	for _, coin := range f.Fees() {
		if coin.IsPositive() {
			denoms[coin.Denom] = true
		}
	}
	if len(f.Fees()) == 0 {
		for _, price := range f.GasPrices() {
			if price.IsPositive() {
				denoms[price.Denom] = true
			}
		}
	}
	config := storetypes.KVGasConfig()
	collector := address.MustLengthPrefix(authtypes.NewModuleAddress(authtypes.FeeCollectorName))
	var allowance uint64
	for denom := range denoms {
		keyBytes := len(banktypes.CreateDenomAddressPrefix(denom)) + len(collector)
		allowance += config.WriteCostFlat + config.WriteCostPerByte*uint64(keyBytes+1)
	}
	return allowance
}

func totalEstimationAllowance(res *txtypes.SimulateResponse, f Factory, msgs ...sdk.Msg) uint64 {
	counter := estimationAllowance(res, msgs...)
	if counter == 0 {
		return 0
	}
	return counter + feeCollectorGasAllowance(f)
}

// adjustedGas uses the decimal adjustment supplied by the caller and rounds
// upward without losing uint64 precision in a float conversion.
func adjustedGas(used, allowance uint64, adjustment float64) (uint64, error) {
	if math.IsNaN(adjustment) || math.IsInf(adjustment, 0) || adjustment < 1 {
		return 0, fmt.Errorf("gas adjustment must be finite and at least 1; got %g", adjustment)
	}
	if math.MaxUint64-used < allowance {
		return 0, fmt.Errorf("gas estimate overflows uint64")
	}
	factor, ok := new(big.Rat).SetString(strconv.FormatFloat(adjustment, 'f', -1, 64))
	if !ok {
		return 0, fmt.Errorf("invalid gas adjustment %g", adjustment)
	}
	product := factor.Mul(factor, new(big.Rat).SetInt(new(big.Int).SetUint64(used+allowance)))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(product.Num(), product.Denom(), remainder)
	if remainder.Sign() > 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsUint64() {
		return 0, fmt.Errorf("adjusted gas estimate overflows uint64")
	}
	return quotient.Uint64(), nil
}

func calculateGas(conn gogogrpc.ClientConn, f Factory, msgs ...sdk.Msg) (*txtypes.SimulateResponse, uint64, error) {
	if _, err := adjustedGas(0, 0, f.GasAdjustment()); err != nil {
		return nil, 0, err
	}
	service := txtypes.NewServiceClient(conn)
	simulate := func(gas uint64) (*txtypes.SimulateResponse, error) {
		bz, err := f.WithGas(gas).BuildSimTx(msgs...)
		if err != nil {
			return nil, err
		}
		res, err := service.Simulate(context.Background(), &txtypes.SimulateRequest{TxBytes: bz})
		if err != nil {
			return nil, fmt.Errorf("gas simulation with limit %d failed: %w", gas, err)
		}
		if res == nil || res.GasInfo == nil {
			return nil, fmt.Errorf("gas simulation returned no gas information")
		}
		return res, nil
	}
	// Bootstrap, then measure fee payment at a provisional limit. Verify the
	// resulting candidate with its own encoded limit and calculated fee.
	candidate := uint64(0)
	for call := 0; call < 4; call++ {
		res, err := simulate(candidate)
		if err != nil {
			return nil, 0, err
		}
		required, err := adjustedGas(res.GasInfo.GasUsed, totalEstimationAllowance(res, f, msgs...), f.GasAdjustment())
		if err != nil {
			return nil, 0, err
		}
		if call >= 2 && required <= candidate {
			return res, candidate, nil
		}
		if required > candidate {
			candidate = required
		}
	}
	return nil, 0, fmt.Errorf("automatic gas estimate did not stabilize after 4 simulations; retry with an explicit --gas limit or a larger --gas-adjustment")
}

func gasEstimateResponse(res *txtypes.SimulateResponse, f Factory, msgs ...sdk.Msg) GasEstimateResponse {
	return GasEstimateResponse{
		GasEstimate: f.Gas(), GasUsed: res.GasInfo.GasUsed,
		EstimationAllowance: totalEstimationAllowance(res, f, msgs...), GasAdjustment: f.GasAdjustment(),
	}
}
