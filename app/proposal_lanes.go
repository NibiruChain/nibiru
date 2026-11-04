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
	// is an oracle price write ([isOraclePriceWriteMsg]), plus earlier
	// transactions from the same allowlisted sender that
	// [splitNonEVMProposalTxs] promotes ahead of them.
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
// Lanes never reorder one signer's transactions, fee payers included, since
// ante checks every tx-level signer's account sequence. Two passes keep that
// order:
//
//  1. Promotion, for allowlisted senders only (ZeroGasActors.Senders): a
//     candidate moves up to the earliest lane of any later candidate that
//     shares an allowlisted signer with it. A feeder that sends any tx at
//     sequence N before its price write at N+1 therefore keeps the price write
//     in [proposalLaneOracle], with the earlier tx ahead of it, instead of
//     pushing the price write behind EVM.
//  2. Demotion, for every signer: a candidate moves down to the latest lane
//     already holding an earlier candidate that shares a signer with it. This
//     covers signers outside the allowlist, such as a fee payer, whose
//     transactions never gain priority. After pass 1 it moves a candidate
//     only when some transaction in its chain of shared signers has a signer
//     outside the allowlist.
//
// Without these passes, a transaction at sequence N+1 could be verified ahead
// of the same account's transaction at N, fail the sequence check, and miss
// the block.
func splitNonEVMProposalTxs(
	decode sdk.TxDecoder,
	candidates [][]byte,
	zeroGasActors sudo.ZeroGasActors,
) (lanes [proposalLaneCount][]nonEVMProposalTx) {
	type classified struct {
		candidate nonEVMProposalTx
		lane      proposalLane
		signers   []string
	}
	allowlisted := make(map[string]struct{}, len(zeroGasActors.Senders))
	for _, sender := range zeroGasActors.Senders {
		allowlisted[sender] = struct{}{}
	}

	txs := make([]classified, 0, len(candidates))
	for _, txBytes := range candidates {
		tx, err := decode(txBytes)
		if err != nil {
			continue
		}
		lane, signers, ok := classifyProposalCandidate(tx, zeroGasActors)
		if !ok || lane == proposalLaneEVM {
			continue
		}
		item := classified{
			candidate: nonEVMProposalTx{txBytes: txBytes},
			lane:      lane,
			signers:   make([]string, len(signers)),
		}
		for i, signer := range signers {
			item.signers[i] = signer.String()
		}
		if gasTx, ok := tx.(interface{ GetGas() uint64 }); ok {
			item.candidate.gasWanted = gasTx.GetGas()
		}
		txs = append(txs, item)
	}

	// Pass 1, backward: earliestLater maps an allowlisted signer to the
	// earliest lane among its later candidates.
	earliestLater := make(map[string]proposalLane)
	for i := len(txs) - 1; i >= 0; i-- {
		for _, signer := range txs[i].signers {
			if _, ok := allowlisted[signer]; !ok {
				continue
			}
			if next, seen := earliestLater[signer]; seen && next < txs[i].lane {
				txs[i].lane = next
			}
		}
		for _, signer := range txs[i].signers {
			if _, ok := allowlisted[signer]; ok {
				earliestLater[signer] = txs[i].lane
			}
		}
	}

	// Pass 2, forward: latestLane maps any signer to the latest lane holding
	// one of its earlier candidates.
	latestLane := make(map[string]proposalLane)
	for i := range txs {
		for _, signer := range txs[i].signers {
			if prev, seen := latestLane[signer]; seen && prev > txs[i].lane {
				txs[i].lane = prev
			}
		}
		for _, signer := range txs[i].signers {
			latestLane[signer] = txs[i].lane
		}
		lanes[txs[i].lane] = append(lanes[txs[i].lane], txs[i].candidate)
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
