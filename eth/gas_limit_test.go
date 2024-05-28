package eth_test

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

// TestGasMeter: Ensures correct behvaior of the `InfiniteGasMeterWithLimit`
// implementation by checking that gas consumption and usage values are correctly
// tracked, that consuming gas within the limit does not cause a panic, and that
// gas refunds work as expected.
func (s *Suite) TestGasMeter() {
	meter := eth.NewInfiniteGasMeter()
	s.Require().Equal(uint64(math.MaxUint64), meter.Limit())
	s.Require().Equal(uint64(math.MaxUint64), meter.GasRemaining())
	s.Require().Equal(uint64(0), meter.GasConsumed())
	s.Require().Equal(uint64(0), meter.GasConsumedToLimit())

	meter.ConsumeGas(10, "consume 10")
	s.Require().Equal(uint64(math.MaxUint64), meter.GasRemaining())
	s.Require().Equal(uint64(10), meter.GasConsumed())
	s.Require().Equal(uint64(10), meter.GasConsumedToLimit())

	// Test RefundGas
	meter.RefundGas(1, "refund 1")
	s.Require().Equal(uint64(math.MaxUint64), meter.GasRemaining())
	s.Require().Equal(uint64(9), meter.GasConsumed())

	// Test IsPastLimit and IsOutOfGas
	s.False(meter.IsPastLimit())
	s.False(meter.IsOutOfGas())

	// Consume large amount fo gas to test overflow handling
	meter.ConsumeGas(sdk.Gas(math.MaxUint64/2), "consume half max uint64")
	s.Require().Panics(func() { meter.ConsumeGas(sdk.Gas(math.MaxUint64/2)+2, "panic") })
	s.Require().Panics(func() { meter.RefundGas(meter.GasConsumed()+1, "refund greater than consumed") })

	// Additional tests for RefundGas
	s.Require().NotPanics(func() {
		meter.RefundGas(meter.GasConsumed(), "refund all")
	})
	s.Require().Equal(uint64(0), meter.GasConsumed())
	s.Require().Panics(func() {
		meter.RefundGas(meter.GasConsumed()+1, "refund more than consumed")
	})
	s.Require().NotPanics(func() { meter.RefundGas(meter.GasConsumed(), "refund all consumed gas") })
	s.Require().Equal(uint64(0), meter.GasConsumed())
	s.Require().Equal(uint64(math.MaxUint64), meter.GasRemaining())

	// Additional tests for IsPastLimit and IsOutOfGas with high gas usage
	s.Equal(uint64(math.MaxUint64), meter.GasRemaining())
	meter.ConsumeGas(sdk.Gas(math.MaxUint64-1), "consume nearly all gas")
	s.Equal(uint64(math.MaxUint64), meter.GasRemaining())
	s.Require().False(meter.IsPastLimit())
	s.Require().False(meter.IsOutOfGas())

	// Test the String method
	expectedString := `InfiniteGasMeter: {"consumed":18446744073709551614,"limit":18446744073709551615}`
	s.Require().Equal(expectedString, meter.String())

	// Test another instance with a specific limit
	meter2 := eth.NewInfiniteGasMeterWithLimit(100)
	s.Require().Equal(uint64(100), meter2.Limit())
	s.Require().Equal(uint64(0), meter2.GasConsumed())

	meter2.ConsumeGas(50, "consume 50")
	s.Require().Equal(uint64(50), meter2.GasConsumed())
	s.False(meter2.IsPastLimit())
	s.False(meter2.IsOutOfGas())

	meter2.ConsumeGas(50, "consume remaining 50")
	s.Require().Equal(uint64(math.MaxUint64), meter2.GasRemaining())
	s.False(meter2.IsPastLimit())
	s.False(meter2.IsOutOfGas())
	s.Require().NotPanics(func() { meter2.ConsumeGas(1, "exceed limit") })

	// Test the String method for the second meter
	expectedString2 := `InfiniteGasMeter: {"consumed":101,"limit":100}`
	s.Require().Equal(expectedString2, meter2.String())
}

func (s *Suite) TestBlockGasLimit() {
	newCtx := func() sdk.Context { return evmtest.NewTestDeps().Ctx }
	tests := []struct {
		name         string
		setupContext func() sdk.Context
		wantGasLimit uint64
	}{
		{
			name: "BlockGasMeter is not nil and has a non-zero limit",
			setupContext: func() sdk.Context {
				ctx := newCtx()
				gasMeter := eth.NewInfiniteGasMeterWithLimit(100)
				ctx = ctx.WithBlockGasMeter(gasMeter)
				return ctx
			},
			wantGasLimit: 100,
		},
		{
			name: "BlockGasMeter is nil and ConsensusParams is nil",
			setupContext: func() sdk.Context {
				ctx := newCtx()
				ctx = ctx.WithConsensusParams(nil)
				return ctx
			},
			wantGasLimit: 0,
		},
		{
			name: "BlockGasMeter is nil and ConsensusParams has Block with MaxGas -1",
			setupContext: func() sdk.Context {
				ctx := newCtx()
				cp := &tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{
						MaxGas: -1,
					},
				}
				ctx = ctx.WithConsensusParams(cp)
				return ctx
			},
			wantGasLimit: math.MaxUint64,
		},
		{
			name: "BlockGasMeter is nil and ConsensusParams has Block with MaxGas > 0",
			setupContext: func() sdk.Context {
				ctx := newCtx()
				cp := &tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{
						MaxGas: 1000,
					},
				}
				ctx = ctx.WithConsensusParams(cp)
				return ctx
			},
			wantGasLimit: 1000,
		},
		{
			name: "BlockGasMeter is nil and ConsensusParams has Block with MaxGas 0",
			setupContext: func() sdk.Context {
				ctx := newCtx()
				cp := &tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{
						MaxGas: 0,
					},
				}
				ctx = ctx.WithConsensusParams(cp)
				return ctx
			},
			wantGasLimit: 0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.setupContext()
			gotGaslimit := eth.BlockGasLimit(ctx)
			s.Require().Equal(tt.wantGasLimit, gotGaslimit)
		})
	}
}
