package app

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
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

// FuzzClassifyProposalCandidates runs the lane classifier and the lane split
// over arbitrary candidate bytes, under the resolved Sai oracle address and
// the message-key fallback. Neither may panic. Every candidate that
// classifies into the oracle lane must be a zero-gas actor tx whose only
// tx-level signer is its first message's sender, every message executing
// the oracle contract under the resolved rule. The split must return each
// decodable non-EVM candidate exactly once, keep relative order within a
// lane, and leave the EVM lane empty.
func FuzzClassifyProposalCandidates(f *testing.F) {
	fx := newClassifyFixture(f)
	encode := MakeEncodingConfig().TxConfig.TxEncoder()
	decode := MakeEncodingConfig().TxConfig.TxDecoder()
	seedTx := func(tx sdk.Tx) []byte {
		txBytes, err := encode(tx)
		require.NoError(f, err)
		return txBytes
	}
	priceWrite := seedTx(fx.build(nil, fx.exec(fx.feeder, fx.oracle, testSetPricesMsg)))
	expiry := seedTx(fx.build(nil, fx.exec(fx.otherFeeder, fx.oracle, testExpiryMsg)))
	sponsored := seedTx(fx.build(fx.payer, fx.exec(fx.feeder, fx.oracle, testSetPricesMsg)))
	twoSenders := seedTx(fx.build(nil,
		fx.exec(fx.feeder, fx.oracle, testSetPricesMsg),
		fx.exec(fx.otherFeeder, fx.oracle, testSetPricesMsg)))
	perpFill := seedTx(fx.build(nil, fx.exec(fx.feeder, fx.perp, testExpiryMsg)))
	trader := seedTx(fx.build(nil, fx.exec(fx.payer, fx.perp, testExpiryMsg)))

	f.Add(false, encodeFuzzTxs())
	f.Add(true, encodeFuzzTxs(perpFill, priceWrite, expiry, trader))
	f.Add(false, encodeFuzzTxs(sponsored, twoSenders, priceWrite))
	f.Add(true, encodeFuzzTxs(trader, []byte("not a tx"), sponsored, expiry))
	f.Add(false, []byte{0xff, 0xff, 0x0a, 0x00})

	f.Fuzz(func(t *testing.T, resolved bool, data []byte) {
		oracle := oracleLaneRule{}
		if resolved {
			oracle.saiOracle = fx.oracle
		}
		candidates := decodeFuzzTxs(data)
		want := make(map[proposalLane][]string)
		for _, txBytes := range candidates {
			tx, err := decode(txBytes)
			if err != nil {
				continue
			}
			lane, signers, ok := classifyProposalCandidate(tx, fx.zeroGasActors, oracle)
			if !ok {
				continue
			}
			require.True(t, lane >= proposalLaneOracle && lane <= proposalLaneDefault,
				"classifier returned lane %d outside the declared lanes", lane)
			if lane == proposalLaneEVM {
				continue
			}
			want[lane] = append(want[lane], string(txBytes))
			if lane != proposalLaneOracle {
				continue
			}
			require.True(t, ante.IsZeroGasActorTx(tx, fx.zeroGasActors))
			require.Len(t, signers, 1)
			require.True(t, signers[0].Equals(tx.GetMsgs()[0].GetSigners()[0]))
			if resolved {
				for _, msg := range tx.GetMsgs() {
					exec, ok := msg.(*wasm.MsgExecuteContract)
					require.True(t, ok)
					require.Equal(t, fx.oracle, exec.Contract)
				}
			}
		}

		lanes := splitNonEVMProposalTxs(decode, candidates, fx.zeroGasActors, oracle)
		require.Empty(t, lanes[proposalLaneEVM])
		var got, all []string
		for _, lane := range lanes {
			var laneTxs []string
			for _, candidate := range lane {
				laneTxs = append(laneTxs, string(candidate.txBytes))
			}
			require.True(t, isSubsequence(laneTxs, candidates),
				"lane reorders its candidates")
			got = append(got, laneTxs...)
		}
		for _, lane := range want {
			all = append(all, lane...)
		}
		// Promotion moves txs between lanes, so compare as multisets.
		require.ElementsMatch(t, all, got)
	})
}

// isSubsequence reports whether sub appears in seq in order.
func isSubsequence(sub []string, seq [][]byte) bool {
	i := 0
	for _, item := range seq {
		if i < len(sub) && sub[i] == string(item) {
			i++
		}
	}
	return i == len(sub)
}
