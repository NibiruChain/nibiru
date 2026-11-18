package app_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
)

// fuzzChooser is a [laneChooser] that draws choices from fuzz input bytes,
// so the fuzzer's byte mutations map to small changes in the generated
// candidates. Once the input is exhausted, every choice is 0.
type fuzzChooser struct {
	data []byte
}

func (c *fuzzChooser) IntN(n int) int {
	if n <= 1 {
		return 0
	}
	var v int
	width := 1
	if n > 256 {
		width = 2
	}
	for range width {
		v <<= 8
		if len(c.data) > 0 {
			v |= int(c.data[0])
			c.data = c.data[1:]
		}
	}
	return v % n
}

// FuzzPrepareProposalLanes derives a candidate list, budgets, allowlist,
// oracle mode, and verifier outcomes from the fuzz input with the generator
// of [TestPrepareProposalLaneProperties], runs PrepareProposal, and checks
// the output with [checkPreparedLanes]: no panic (the handler would return
// an empty block, which [runLaneCase] rejects while an EVM tx fits),
// sequence order in a block-order replay, verification order equal to block
// order, lane order including the oracle-lane rule, and the byte and gas
// budgets. It uses the sequence-checking verifier stub, which lets it vary
// max gas and verifier rejections freely;
// [FuzzPrepareProposalLanesRealProcess] covers the real ante handler and the
// stock ProcessProposal.
func FuzzPrepareProposalLanes(f *testing.F) {
	// The test app keeps its Wasm VM under ./data and locks it. Fuzz workers
	// are separate processes, so each needs its own directory.
	f.Chdir(f.TempDir())
	deps := evmtest.NewTestDeps()
	world := newLanePropertyWorld(f, &deps)

	f.Add([]byte{})
	f.Add([]byte{8, 0, 0, 0, 0, 1, 1, 1, 1})
	f.Add([]byte("\x10\x00\x15\x02\x00\x00\x00\x09\x30\x00\x00\x00\x40\x01" +
		"\x00\x00\x00\x50\x00\x00\x00\x00\x00\x00\x01\x00\x63"))
	f.Add([]byte("oracle lanes from fuzz bytes, with promotion and caps"))

	var stats laneCheckStats
	f.Fuzz(func(t *testing.T, data []byte) {
		choose := &fuzzChooser{data: data}
		runLaneCase(t, &deps, world, choose, choose.IntN(40), "fuzz", &stats)
	})
}

// FuzzPrepareProposalLanesRealProcess derives a candidate list of really
// signed txs, with fee payers, co-signers, bad signatures, and sequence gaps,
// and a MaxTxBytes from the fuzz input, then requires that the block the
// app's PrepareProposal builds with the real ante handler is accepted by the
// app's stock ProcessProposal and passes [checkPreparedLanes]. The first
// input byte selects the resolved Sai oracle or the message-key fallback.
func FuzzPrepareProposalLanesRealProcess(f *testing.F) {
	// See FuzzPrepareProposalLanes on the working directory.
	f.Chdir(f.TempDir())
	fixtures := [2]*laneE2EFixture{
		newLaneE2EFixture(f, false), newLaneE2EFixture(f, true),
	}

	f.Add([]byte{})
	f.Add([]byte{1, 12, 0, 0, 1, 0, 40, 0, 50, 0, 90, 0, 1, 2})
	f.Add([]byte("oracle lanes from fuzz bytes, with promotion and caps"))

	var stats laneCheckStats
	f.Fuzz(func(t *testing.T, data []byte) {
		choose := &fuzzChooser{data: data}
		fixture := fixtures[choose.IntN(2)]
		runRealLaneCase(t, fixture, choose, choose.IntN(30), "fuzz", &stats)
	})
}
