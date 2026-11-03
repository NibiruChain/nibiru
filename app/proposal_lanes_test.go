package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// laneFixture holds one encoded transaction per lane plus the zero-gas set
// that classifies them.
type laneFixture struct {
	deps      evmtest.TestDeps
	actors    sudo.ZeroGasActors
	oracle    []byte // feeder -> oracle contract, set_prices
	oracle2   []byte // feeder -> oracle contract, set_price
	zeroGas   []byte // keeper bot -> perp contract, non-price execute
	evm       []byte
	cosmos    []byte // trader -> oracle contract (sender not zero-gas)
	decodeTxs sdk.TxDecoder
}

func buildWasmExecTx(
	t *testing.T, deps *evmtest.TestDeps, sender string, execs ...[2]string,
) []byte {
	t.Helper()
	msgs := make([]sdk.Msg, 0, len(execs))
	for _, exec := range execs {
		msgs = append(msgs, &wasm.MsgExecuteContract{
			Sender:   sender,
			Contract: exec[0],
			Msg:      wasm.RawContractMessage(exec[1]),
		})
	}
	builder := deps.App.GetTxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msgs...))
	builder.SetGasLimit(1_000_000)
	txBytes, err := deps.App.GetTxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

func newLaneFixture(t *testing.T) laneFixture {
	deps := evmtest.NewTestDeps()
	feeder := evmtest.NewEthPrivAcc().NibiruAddr.String()
	keeperBot := evmtest.NewEthPrivAcc().NibiruAddr.String()
	trader := evmtest.NewEthPrivAcc().NibiruAddr.String()
	oracleContract := evmtest.NewEthPrivAcc().NibiruAddr.String()
	perpContract := evmtest.NewEthPrivAcc().NibiruAddr.String()
	actors := sudo.ZeroGasActors{
		Senders:   []string{feeder, keeperBot},
		Contracts: []string{oracleContract, perpContract},
	}
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), actors)

	msg := evmtest.HappyTransferTx(&deps, 0)
	evmTx := evmtest.BuildTx(&deps, true, msg.GetGas(), nil, msg)
	evmBytes, err := deps.App.GetTxConfig().TxEncoder()(evmTx)
	require.NoError(t, err)

	return laneFixture{
		deps:   deps,
		actors: actors,
		oracle: buildWasmExecTx(t, &deps, feeder,
			[2]string{oracleContract, `{"set_prices":{"prices":[]}}`}),
		oracle2: buildWasmExecTx(t, &deps, feeder,
			[2]string{oracleContract, `{"set_price":{"pair":"btc","price":"1"}}`}),
		zeroGas: buildWasmExecTx(t, &deps, keeperBot,
			[2]string{perpContract, `{"liquidate":{}}`}),
		evm: evmBytes,
		cosmos: buildWasmExecTx(t, &deps, trader,
			[2]string{oracleContract, `{"set_prices":{"prices":[]}}`}),
		decodeTxs: deps.App.GetTxConfig().TxDecoder(),
	}
}

func (f laneFixture) lane(t *testing.T, txBytes []byte) app.ProposalLane {
	tx, err := f.decodeTxs(txBytes)
	require.NoError(t, err)
	return app.ClassifyProposalTx(tx, f.actors)
}

func TestClassifyProposalTx(t *testing.T) {
	f := newLaneFixture(t)
	feeder, keeperBot := f.actors.Senders[0], f.actors.Senders[1]
	oracleContract, perpContract := f.actors.Contracts[0], f.actors.Contracts[1]

	require.Equal(t, app.LaneOracle, f.lane(t, f.oracle))
	require.Equal(t, app.LaneOracle, f.lane(t, f.oracle2))
	require.Equal(t, app.LaneZeroGas, f.lane(t, f.zeroGas))
	require.Equal(t, app.LaneEVM, f.lane(t, f.evm))
	require.Equal(t, app.LaneCosmos, f.lane(t, f.cosmos))

	for name, tc := range map[string]struct {
		txBytes []byte
		want    app.ProposalLane
	}{
		"batch of price writes": {buildWasmExecTx(t, &f.deps, feeder,
			[2]string{oracleContract, `{"set_prices":{}}`},
			[2]string{perpContract, `{"set_price":{}}`}), app.LaneOracle},
		"price write plus other execute": {buildWasmExecTx(t, &f.deps, keeperBot,
			[2]string{oracleContract, `{"set_prices":{}}`},
			[2]string{perpContract, `{"liquidate":{}}`}), app.LaneZeroGas},
		"two top-level keys": {buildWasmExecTx(t, &f.deps, feeder,
			[2]string{oracleContract, `{"set_prices":{},"x":1}`}), app.LaneZeroGas},
		"not an object": {buildWasmExecTx(t, &f.deps, feeder,
			[2]string{oracleContract, `"set_prices"`}), app.LaneZeroGas},
		"nested key only": {buildWasmExecTx(t, &f.deps, feeder,
			[2]string{oracleContract, `{"x":{"set_prices":{}}}`}), app.LaneZeroGas},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, f.lane(t, tc.txBytes))
		})
	}
}

func TestVerifyProposalLaneOrder(t *testing.T) {
	f := newLaneFixture(t)
	mono := app.LaneOrderConfig{Rule: app.LaneOrderMonotonic}
	first := app.LaneOrderConfig{Rule: app.LaneOrderOracleFirst}

	for _, tc := range []struct {
		name      string
		txs       [][]byte
		monoOK    bool
		oracleFOK bool
	}{
		{"empty", nil, true, true},
		{"all lanes in order",
			[][]byte{f.oracle, f.oracle2, f.zeroGas, f.evm, f.evm, f.cosmos}, true, true},
		{"skipped lanes", [][]byte{f.oracle, f.cosmos}, true, true},
		{"no oracle write", [][]byte{f.zeroGas, f.evm, f.cosmos}, true, true},
		{"only evm", [][]byte{f.evm, f.evm}, true, true},
		{"oracle after evm", [][]byte{f.evm, f.oracle}, false, false},
		{"oracle after cosmos", [][]byte{f.cosmos, f.oracle, f.evm}, false, false},
		{"oracle after zero-gas", [][]byte{f.zeroGas, f.oracle}, false, false},
		{"cosmos before evm", [][]byte{f.oracle, f.cosmos, f.evm}, false, true},
		{"zero-gas after evm", [][]byte{f.evm, f.zeroGas}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errMono := app.VerifyProposalLaneOrder(f.decodeTxs, tc.txs, f.actors, mono)
			errFirst := app.VerifyProposalLaneOrder(f.decodeTxs, tc.txs, f.actors, first)
			require.Equal(t, tc.monoOK, errMono == nil, "monotonic: %v", errMono)
			require.Equal(t, tc.oracleFOK, errFirst == nil, "oracle-first: %v", errFirst)
		})
	}

	t.Run("undecodable tx rejected", func(t *testing.T) {
		txs := [][]byte{f.oracle, []byte("not a tx")}
		require.ErrorContains(t,
			app.VerifyProposalLaneOrder(f.decodeTxs, txs, f.actors, mono), "undecodable")
	})

	t.Run("empty zero-gas set sends price writes to cosmos lane", func(t *testing.T) {
		txs := [][]byte{f.evm, f.oracle}
		require.Error(t, app.VerifyProposalLaneOrder(f.decodeTxs, txs, f.actors, mono))
		require.NoError(t,
			app.VerifyProposalLaneOrder(f.decodeTxs, txs, sudo.ZeroGasActors{}, mono))
	})

	t.Run("oracle lane cap", func(t *testing.T) {
		capped := app.LaneOrderConfig{Rule: app.LaneOrderMonotonic, MaxOracleLaneTxs: 1}
		require.NoError(t, app.VerifyProposalLaneOrder(
			f.decodeTxs, [][]byte{f.oracle, f.evm}, f.actors, capped))
		require.ErrorContains(t, app.VerifyProposalLaneOrder(
			f.decodeTxs, [][]byte{f.oracle, f.oracle2}, f.actors, capped), "exceeds")
	})
}

func TestLaneOrderActiveAt(t *testing.T) {
	require.False(t, app.LaneOrderActiveAt(0, 100), "upgrade not run")
	require.False(t, app.LaneOrderActiveAt(100, 100), "upgrade height itself")
	require.True(t, app.LaneOrderActiveAt(100, 101))
}

// TestLaneOrderProcessProposalHandler proves the gate: before activation a
// misordered proposal reaches next unchanged; after activation it is rejected
// without calling next.
func TestLaneOrderProcessProposalHandler(t *testing.T) {
	f := newLaneFixture(t)
	nextCalls := 0
	next := func(sdk.Context, abci.RequestProcessProposal) abci.ResponseProcessProposal {
		nextCalls++
		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
	}
	active := false
	handler := app.NewLaneOrderProcessProposalHandler(
		f.decodeTxs,
		f.deps.App.SudoKeeper.GetZeroGasActors,
		func(sdk.Context) bool { return active },
		app.DefaultLaneOrderConfig,
		next,
	)
	misordered := abci.RequestProcessProposal{Height: 5, Txs: [][]byte{f.evm, f.oracle}}
	ordered := abci.RequestProcessProposal{Height: 5, Txs: [][]byte{f.oracle, f.evm}}

	resp := handler(f.deps.Ctx(), misordered)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, resp.Status)
	require.Equal(t, 1, nextCalls)

	active = true
	resp = handler(f.deps.Ctx(), misordered)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, resp.Status)
	require.Equal(t, 1, nextCalls)

	resp = handler(f.deps.Ctx(), ordered)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, resp.Status)
	require.Equal(t, 2, nextCalls)
}

// TestPrepareProposalSatisfiesLaneOrder proves the proposer and the verifier
// agree: CometBFT candidates in adversarial order (zero-gas before oracle,
// Cosmos first) come out of PrepareProposal in an order ProcessProposal
// accepts. A mismatch here would halt the chain once enforcement activates.
func TestPrepareProposalSatisfiesLaneOrder(t *testing.T) {
	f := newLaneFixture(t)
	account := evmstate.NewEmptyAccount()
	require.NoError(t,
		f.deps.EvmKeeper.SetAccount(f.deps.Ctx(), f.deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &f.deps, 0, 100)

	handler := app.NewEVMPrepareProposalHandler(f.deps.App, new(acceptingPrepareVerifier))
	ctx := f.deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	resp := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{f.cosmos, f.zeroGas, f.oracle, f.oracle2},
	})
	require.Equal(t,
		[][]byte{f.oracle, f.oracle2, f.zeroGas, evmTx, f.cosmos}, resp.Txs)
	require.NoError(t, app.VerifyProposalLaneOrder(
		f.decodeTxs, resp.Txs, f.actors, app.DefaultLaneOrderConfig))
}
