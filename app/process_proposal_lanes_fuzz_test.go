package app_test

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"

	"github.com/NibiruChain/nibiru/v2/app"
	srvconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
)

// encodeFuzzTxs joins txs as a fuzz input: each tx is a 2-byte big-endian
// length followed by its bytes. See [decodeFuzzTxs].
func encodeFuzzTxs(txs ...[]byte) []byte {
	var out []byte
	for _, txBytes := range txs {
		out = binary.BigEndian.AppendUint16(out, uint16(len(txBytes)))
		out = append(out, txBytes...)
	}
	return out
}

// decodeFuzzTxs splits a fuzz input into txs as [encodeFuzzTxs] joins them.
// A length past the end takes the rest of the input, so every input decodes.
func decodeFuzzTxs(data []byte) [][]byte {
	var txs [][]byte
	for len(data) >= 2 {
		size := int(binary.BigEndian.Uint16(data))
		data = data[2:]
		size = min(size, len(data))
		txs = append(txs, data[:size])
		data = data[size:]
	}
	return txs
}

// FuzzVerifyProposalLaneRules runs the lane rule check of
// [app.NewLaneOrderProcessProposalHandler] over arbitrary proposal txs. The
// check must never panic out of the handler and must decide the same way
// twice on the same input. Seeds include price writes, keeper fills, trader
// calls, a sponsored price write, an EVM tx, an invalid sender, and garbage.
func FuzzVerifyProposalLaneRules(f *testing.F) {
	// The test app keeps its Wasm VM under ./data and locks it. Fuzz workers
	// are separate processes, so each needs its own directory.
	f.Chdir(f.TempDir())
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(f, deps.EvmKeeper.SetAccount(
		deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(f, &deps, 0, 100)
	a := setSaiProposalActors(&deps)
	deps.App.SetLaneRulesMode(srvconfig.LaneRulesOn)
	ctx := laneRulesCtx(&deps, 1_000_000, -1)
	handler := app.NewLaneOrderProcessProposalHandler(
		deps.App, acceptAllProcessProposal)

	setPrices := buildSequencedProposalWasmTx(f, &deps, nil, []uint64{0},
		proposalWasmExec(a.feeder, a.oracle, setPricesMsg))
	keeperFill := buildProposalWasmTx(f, &deps,
		proposalWasmExec(a.keeper, a.perp, triggerFillMsg))
	traderTx := buildProposalWasmTx(f, &deps,
		proposalWasmExec(a.trader, a.perp, triggerFillMsg))
	sponsored := buildSequencedProposalWasmTx(f, &deps,
		evmtest.NewEthPrivAcc().NibiruAddr, []uint64{1, 0},
		proposalWasmExec(a.feeder, a.oracle, setPriceMsg))
	invalidSender := buildProposalWasmTx(f, &deps,
		proposalWasmExec("not-a-bech32-address", a.oracle, setPricesMsg))

	f.Add(encodeFuzzTxs())
	f.Add(encodeFuzzTxs(setPrices, keeperFill, evmTx, traderTx))
	f.Add(encodeFuzzTxs(traderTx, setPrices, sponsored))
	f.Add(encodeFuzzTxs(invalidSender, setPrices))
	f.Add(encodeFuzzTxs(evmTx, setPrices, []byte("not a tx")))
	f.Add([]byte{0xff, 0xff, 0x0a, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		req := abci.RequestProcessProposal{
			Height: laneRulesDoneHeight + 1,
			Txs:    decodeFuzzTxs(data),
		}
		first := handler(ctx, req)
		second := handler(ctx, req)
		require.Equal(t, first.Status, second.Status)
	})
}
