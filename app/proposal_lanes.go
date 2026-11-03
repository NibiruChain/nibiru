package app

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/evm"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authsigning "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/signing"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// proposalLane is the block section a transaction is proposed in. Lanes are
// declared in proposal order, so a lower value is proposed earlier: every
// selected transaction of a lane precedes every selected transaction of a
// later lane. See [classifyProposalTx] and [splitNonEVMProposalTxs].
type proposalLane uint8

const (
	// proposalLaneOracle holds zero-gas actor transactions whose every message
	// is an oracle price write ([isOraclePriceWriteMsg]).
	proposalLaneOracle proposalLane = iota
	// proposalLaneZeroGas holds the remaining zero-gas actor transactions
	// ([ante.IsZeroGasActorTx]), such as keeper and admin calls.
	proposalLaneZeroGas
	// proposalLaneEVM holds Ethereum transactions. PrepareProposal fills it
	// from [evm.Mempool], not from RequestPrepareProposal.Txs.
	proposalLaneEVM
	// proposalLaneDefault holds every other non-EVM transaction.
	proposalLaneDefault

	proposalLaneCount
)

// classifyProposalTx assigns tx to its [proposalLane]. zeroGasActors is the
// x/sudo allowlist from the proposal's state.
func classifyProposalTx(tx sdk.Tx, zeroGasActors sudo.ZeroGasActors) proposalLane {
	switch {
	case evm.IsEthTx(tx):
		return proposalLaneEVM
	case !ante.IsZeroGasActorTx(tx, zeroGasActors):
		return proposalLaneDefault
	case isOraclePriceWriteTx(tx):
		return proposalLaneOracle
	default:
		return proposalLaneZeroGas
	}
}

// isOraclePriceWriteTx reports whether tx has at least one message and every
// message is a Wasm execute carrying an oracle price write.
func isOraclePriceWriteTx(tx sdk.Tx) bool {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return false
	}
	for _, msg := range msgs {
		msgExec, ok := msg.(*wasm.MsgExecuteContract)
		if !ok || !isOraclePriceWriteMsg(msgExec.Msg) {
			return false
		}
	}
	return true
}

// oraclePriceWriteMsgKeys are the externally tagged (snake_case serde)
// execute variants with which the Sai oracle contract records prices.
var oraclePriceWriteMsgKeys = map[string]struct{}{
	"set_prices": {},
	"set_price":  {},
}

// isOraclePriceWriteMsg reports whether a Wasm execute message is a JSON
// object with exactly one top-level key naming an oracle price write. It
// matches the message rather than a contract address, so no per-network
// configuration is needed. Malformed JSON, trailing data, or extra keys return
// false. The variant payload is not validated; the contract does that on
// execution.
func isOraclePriceWriteMsg(raw []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return false
	}
	keyTok, err := dec.Token()
	if err != nil {
		return false
	}
	key, ok := keyTok.(string)
	if !ok {
		return false
	}
	if _, ok := oraclePriceWriteMsgKeys[key]; !ok {
		return false
	}
	var value json.RawMessage
	if err := dec.Decode(&value); err != nil {
		return false
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return false
	}
	_, err = dec.Token()
	return err == io.EOF
}

// nonEVMProposalTx is a decoded non-EVM candidate from
// RequestPrepareProposal.Txs with its original outer bytes.
type nonEVMProposalTx struct {
	txBytes   []byte
	gasWanted uint64
}

// splitNonEVMProposalTxs decodes the CometBFT candidates once and groups the
// non-EVM transactions by [classifyProposalTx], preserving relative order
// within each lane. Undecodable and EVM candidates are dropped, so the
// [proposalLaneEVM] entry is always empty. A candidate whose classification
// panics is also dropped ([classifyProposalCandidate]).
//
// Lanes never reorder one signer's transactions, fee payers included: a
// candidate is demoted to the latest lane already holding an earlier candidate
// that shares a signer with it. Without this, a zero-gas transaction at account sequence N+1 could
// be verified ahead of the same account's default-lane transaction at N, fail
// the sequence check, and miss the block.
func splitNonEVMProposalTxs(
	decode sdk.TxDecoder,
	candidates [][]byte,
	zeroGasActors sudo.ZeroGasActors,
) (lanes [proposalLaneCount][]nonEVMProposalTx) {
	// latestLane maps a signer address to the latest lane holding one of its
	// candidates so far. Lane values increase in proposal order.
	latestLane := make(map[string]proposalLane)
	for _, txBytes := range candidates {
		tx, err := decode(txBytes)
		if err != nil {
			continue
		}
		lane, signers, ok := classifyProposalCandidate(tx, zeroGasActors)
		if !ok || lane == proposalLaneEVM {
			continue
		}
		for _, signer := range signers {
			if prev, seen := latestLane[string(signer)]; seen && prev > lane {
				lane = prev
			}
		}
		for _, signer := range signers {
			latestLane[string(signer)] = lane
		}
		candidate := nonEVMProposalTx{txBytes: txBytes}
		if gasTx, ok := tx.(interface{ GetGas() uint64 }); ok {
			candidate.gasWanted = gasTx.GetGas()
		}
		lanes[lane] = append(lanes[lane], candidate)
	}
	return lanes
}

// classifyProposalCandidate returns the [classifyProposalTx] lane of a decoded
// but unverified candidate and, for non-EVM transactions, its tx-level
// signers: the message signers plus the fee payer, whose sequences ante
// checks. ok is false when classification panics, for example when
// GetSigners meets an invalid bech32 sender. Such a candidate cannot pass
// signature verification, so it is dropped rather than placed in a lane; the
// recover keeps one malformed candidate from reaching the handler-level
// recover, which would empty the whole proposal.
func classifyProposalCandidate(
	tx sdk.Tx,
	zeroGasActors sudo.ZeroGasActors,
) (lane proposalLane, signers []sdk.AccAddress, ok bool) {
	defer func() {
		if recover() != nil {
			lane, signers, ok = proposalLaneDefault, nil, false
		}
	}()
	lane = classifyProposalTx(tx, zeroGasActors)
	if lane == proposalLaneEVM {
		return lane, nil, true
	}
	// Ante checks and increments the sequence of every tx-level signer,
	// which includes a fee payer that signs no message.
	if sigTx, isSigTx := tx.(authsigning.SigVerifiableTx); isSigTx {
		return lane, sigTx.GetSigners(), true
	}
	for _, msg := range tx.GetMsgs() {
		signers = append(signers, msg.GetSigners()...)
	}
	return lane, signers, true
}
