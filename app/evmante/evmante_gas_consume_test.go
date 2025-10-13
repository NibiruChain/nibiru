package evmante_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

func (s *TestSuite) TestEthAnteBlockGasMeter() {
	type TestCase struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx
		maxGasWanted  uint64
		wantErrSubstr string
	}

	const (
		blkOK    uint64 = 10_000_000
		blkSmall uint64 = 50_000
		blkExact uint64 = 1_500_000
	)

	makeMeter := func(limit uint64) sdk.GasMeter { return eth.NewInfiniteGasMeterWithLimit(limit) }

	withConsensus := func(maxGas int64) func(ctx sdk.Context) sdk.Context {
		return func(ctx sdk.Context) sdk.Context {
			ctx.ConsensusParams()
			return ctx.WithConsensusParams(&tmproto.ConsensusParams{
				Block: &tmproto.BlockParams{MaxGas: maxGas},
			})
		}
	}

	testCases := []TestCase{
		{
			name: "meter limit: tx gas ≤ meter -> ok",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps)
				ctx := sdb.Ctx().WithIsCheckTx(true).WithBlockGasMeter(makeMeter(blkOK))
				sdb.SetCtx(ctx)
				return tx
			},
			maxGasWanted: 0,
		},
		{
			name: "meter limit: tx gas > meter -> error with meter limit",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps) // tx gas assumed > blkSmall
				ctx := sdb.Ctx().WithIsCheckTx(true).WithBlockGasMeter(makeMeter(blkSmall))
				sdb.SetCtx(ctx)
				return tx
			},
			maxGasWanted:  0,
			wantErrSubstr: "exceeds block gas limit (50000)",
		},
		{
			name: "consensus: MaxGas = -1 (unlimited) -> ok",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps)
				ctx := sdb.Ctx()
				ctx = withConsensus(-1)(ctx)                         // unlimited by consensus
				ctx = ctx.WithIsCheckTx(true).WithBlockGasMeter(nil) // ensure meter doesn’t override
				sdb.SetCtx(ctx)
				return tx
			},
			maxGasWanted: 0,
		},
		{
			name: "consensus: MaxGas = 0 -> error",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps)
				ctx := sdb.Ctx()
				ctx = withConsensus(0)(ctx)
				ctx = ctx.WithIsCheckTx(true).WithBlockGasMeter(nil)
				sdb.SetCtx(ctx)
				return tx
			},
			maxGasWanted:  0,
			wantErrSubstr: "exceeds block gas limit (0)",
		},
		{
			name: "consensus: positive MaxGas, tx gas ≤ limit -> ok",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps) // ≤ blkExact
				ctx := sdb.Ctx()
				ctx = withConsensus(int64(blkExact))(ctx)
				ctx = ctx.WithIsCheckTx(true).WithBlockGasMeter(nil)
				sdb.SetCtx(ctx)
				return tx
			},
			maxGasWanted: 0,
		},
		{
			name: "consensus: positive MaxGas, tx gas > limit -> error",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps) // > blkSmall
				ctx := sdb.Ctx()
				ctx = withConsensus(int64(blkSmall))(ctx)
				ctx = ctx.WithIsCheckTx(true).WithBlockGasMeter(nil)
				sdb.SetCtx(ctx)
				return tx
			},
			maxGasWanted:  0,
			wantErrSubstr: "exceeds block gas limit (50000)",
		},
		{
			name: "CheckTx: maxGasWanted caps gasWanted; cap ≤ block -> ok",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps) // tx gas ≥ blkExact
				ctx := sdb.Ctx().WithIsCheckTx(true).WithBlockGasMeter(makeMeter(blkExact))
				sdb.SetCtx(ctx)
				return tx
			},
			maxGasWanted: blkExact, // gasWanted=min(txGas, blkExact)=blkExact
		},
		{
			name: "CheckTx: maxGasWanted > block -> error comparing cap to block",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps)
				ctx := sdb.Ctx().WithIsCheckTx(true).WithBlockGasMeter(makeMeter(blkSmall))
				sdb.SetCtx(ctx)
				return tx
			},
			maxGasWanted:  blkSmall + 1, // gasWanted=blkSmall+1 > block
			wantErrSubstr: "exceeds block gas limit (50000)",
		},
		{
			name: "DeliverTx: ignores maxGasWanted; compares tx gas vs block",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps) // tx gas > blkSmall
				ctx := sdb.Ctx().WithIsCheckTx(false).WithBlockGasMeter(makeMeter(blkSmall))
				sdb.SetCtx(ctx)
				return tx
			},
			maxGasWanted:  blkOK, // ignored in DeliverTx
			wantErrSubstr: "exceeds block gas limit (50000)",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			sdb := deps.NewStateDB()

			tx := tc.beforeTxSetup(&deps, sdb)
			sdb.Commit()

			err := evmante.EthAnteBlockGasMeter(
				sdb,
				sdb.Keeper(),
				tx,
				false,
				AnteOptionsForTests{MaxTxGasWanted: tc.maxGasWanted},
			)
			if tc.wantErrSubstr != "" {
				s.Require().ErrorContains(err, tc.wantErrSubstr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestAnteDecEthGasConsume() {
	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx
		wantErr       string
		maxGasWanted  uint64
		gasMeter      sdk.GasMeter
	}{
		{
			name: "happy: sender with funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				gasLimit := happyGasLimit()
				balance := evm.NativeToWei(new(big.Int).Add(gasLimit, big.NewInt(100)))
				AddBalanceSigned(sdb, deps.Sender.EthAddr, balance)
				return evmtest.HappyCreateContractTx(deps)
			},
			wantErr:      "",
			gasMeter:     eth.NewInfiniteGasMeterWithLimit(happyGasLimit().Uint64()),
			maxGasWanted: 0,
		},
		{
			name: "happy: is recheck tx",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				sdb.SetCtx(
					sdb.Ctx().WithIsReCheckTx(true),
				)
				return evmtest.HappyCreateContractTx(deps)
			},
			gasMeter: eth.NewInfiniteGasMeterWithLimit(0),
			wantErr:  "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			sdb := deps.NewStateDB()

			tx := tc.beforeTxSetup(&deps, sdb)
			sdb.Commit()

			deps.SetCtx(deps.Ctx().
				WithIsCheckTx(true).
				WithBlockGasMeter(tc.gasMeter),
			)

			simulate := false
			err := evmante.EthAnteGasWanted(
				sdb,
				sdb.Keeper(),
				tx,
				simulate,
				AnteOptionsForTests{MaxTxGasWanted: tc.maxGasWanted},
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}
