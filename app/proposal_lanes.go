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
	// proposalLaneOracle holds zero-gas actor transactions that are oracle
	// price writes ([isOraclePriceWriteTx]): every message is a price write
	// and the first message's sender is the only tx-level signer. It also
	// holds earlier transactions sharing a signer with them, which
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

// isOraclePriceWriteTx reports whether tx is an oracle price write: it has
// at least one message, every message is a Wasm execute carrying an oracle
// price write ([isOraclePriceWriteMsg]), and its only tx-level signer is the
// first message's sender. A price write with a co-signer, a message from
// another sender, or a distinct fee payer is not a price write; it is
// classified by the remaining rules, normally into [proposalLaneZeroGas].
//
// The sole-signer condition keeps [promoteProposalLanes] from following a
// price write to accounts other than its sender. Without it, any account that
// co-signs or pays for a price write would have its earlier transactions, and
// transitively those of every account sharing a transaction with it, admitted
// ahead of the block's price writes.
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
	senders := msgs[0].GetSigners()
	signers := proposalTxSigners(tx)
	return len(senders) == 1 && len(signers) == 1 &&
		signers[0].Equals(senders[0])
}

// proposalTxSigners returns the tx-level signers of a non-EVM transaction:
// the message signers plus the fee payer, whose sequences ante checks. It
// panics, through GetSigners, on an invalid bech32 signer or fee payer.
func proposalTxSigners(tx sdk.Tx) (signers []sdk.AccAddress) {
	// Ante checks and increments the sequence of every tx-level signer,
	// which includes a fee payer that signs no message.
	if sigTx, isSigTx := tx.(authsigning.SigVerifiableTx); isSigTx {
		return sigTx.GetSigners()
	}
	for _, msg := range tx.GetMsgs() {
		signers = append(signers, msg.GetSigners()...)
	}
	return signers
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
	// entry is the candidate's own lane from [classifyProposalTx], before
	// promotion, and its tx-level signers.
	entry proposalLaneEntry
}

// splitNonEVMProposalTxs decodes the CometBFT candidates once and groups the
// non-EVM transactions by [classifyProposalTx], preserving relative order
// within each lane. Undecodable and EVM candidates are dropped, so the
// [proposalLaneEVM] entry is always empty. A candidate whose classification
// panics is also dropped ([classifyProposalCandidate]).
//
// Lanes never reorder one signer's transactions, fee payers included, since
// ante checks every tx-level signer's account sequence. Instead of pushing a
// later transaction back, the split promotes earlier ones: walking the
// candidates backward, each moves up to the earliest lane of any later
// candidate that shares a signer with it. A fast-lane transaction therefore
// keeps its lane, and same-signer transactions that precede it ride ahead of
// it. A feeder that sends any tx at sequence N before its price write at N+1
// keeps the price write in [proposalLaneOracle], with the earlier tx first.
//
// Promotion follows every shared signer, so a co-signer or fee payer of a
// fast-lane transaction also has its earlier transactions promoted. That
// requires an allowlisted sender to sign a transaction naming that account,
// and it is preferred to delaying the allowlisted sender's transaction behind
// EVM. A price write has its sender as its only signer
// ([isOraclePriceWriteTx]), so promotion into [proposalLaneOracle] reaches
// only the earlier transactions of price-write senders and of accounts that
// share an earlier transaction with them.
//
// After the backward pass, lanes are non-decreasing along every signer's
// transactions in candidate order, so no transaction at sequence N+1 is
// verified ahead of the same account's transaction at N.
func splitNonEVMProposalTxs(
	decode sdk.TxDecoder,
	candidates [][]byte,
	zeroGasActors sudo.ZeroGasActors,
) (lanes [proposalLaneCount][]nonEVMProposalTx) {
	items := make([]nonEVMProposalTx, 0, len(candidates))
	entries := make([]proposalLaneEntry, 0, len(candidates))
	for _, txBytes := range candidates {
		tx, err := decode(txBytes)
		if err != nil {
			continue
		}
		lane, signers, ok := classifyProposalCandidate(tx, zeroGasActors)
		if !ok || lane == proposalLaneEVM {
			continue
		}
		item := nonEVMProposalTx{
			txBytes: txBytes,
			entry:   newProposalLaneEntry(lane, signers),
		}
		if gasTx, ok := tx.(interface{ GetGas() uint64 }); ok {
			item.gasWanted = gasTx.GetGas()
		}
		items = append(items, item)
		entries = append(entries, item.entry)
	}

	promoteProposalLanes(entries)
	for i, item := range items {
		lanes[entries[i].lane] = append(lanes[entries[i].lane], item)
	}
	return lanes
}

// proposalLaneEntry is one transaction's [proposalLane] and tx-level signers
// as bech32 strings, the input to [promoteProposalLanes].
type proposalLaneEntry struct {
	lane    proposalLane
	signers []string
}

// newProposalLaneEntry returns a [proposalLaneEntry] for a classified
// transaction.
func newProposalLaneEntry(
	lane proposalLane,
	signers []sdk.AccAddress,
) proposalLaneEntry {
	entry := proposalLaneEntry{lane: lane, signers: make([]string, len(signers))}
	for i, signer := range signers {
		entry.signers[i] = signer.String()
	}
	return entry
}

// promoteProposalLanes is the backward promotion pass shared by
// PrepareProposal ([splitNonEVMProposalTxs]) and ProcessProposal
// ([proposalBlockLanes]). Walking entries backward, each moves up to the
// earliest lane of any later entry that shares a signer with it. Afterward,
// lanes are non-decreasing along every signer's entries, so grouping entries
// by lane with a stable order never reorders one signer's transactions.
//
// [proposalLaneEVM] entries neither promote nor are promoted: EVM senders are
// tracked by the EVM mempool, not by Cosmos signers.
func promoteProposalLanes(entries []proposalLaneEntry) {
	// earliestLater maps a signer to the lane of its nearest later entry,
	// which after promotion is the earliest lane of all of them. It is only
	// read by key, so map iteration order never affects the result.
	earliestLater := make(map[string]proposalLane)
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].lane == proposalLaneEVM {
			continue
		}
		for _, signer := range entries[i].signers {
			if next, seen := earliestLater[signer]; seen && next < entries[i].lane {
				entries[i].lane = next
			}
		}
		for _, signer := range entries[i].signers {
			earliestLater[signer] = entries[i].lane
		}
	}
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
	return lane, proposalTxSigners(tx), true
}
