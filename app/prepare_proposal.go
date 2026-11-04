package app

import (
	"bytes"
	"container/heap"
	"errors"
	"math"
	"sort"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// prepareProposalTxVerifier verifies original outer transaction bytes during
// PrepareProposal without decoding and re-encoding them. Byte identity matters
// for EVM transactions because ante validation populates [evm.MsgEthereumTx.From]
// on the decoded object.
type prepareProposalTxVerifier interface {
	PrepareProposalVerifyTxBytes(txBytes []byte) error
}

// proposalChain is a contiguous executable prefix for one sender: the state
// nonce chain entry at the committed state nonce and any consecutive later
// nonces. [proposalChain.index] advances only after the current head is selected
// so the handler never skips one sender nonce to include a later nonce.
type proposalChain struct {
	txs   []evm.MempoolTx
	index int
}

// current returns the next executable transaction in the chain.
func (chain *proposalChain) current() evm.MempoolTx {
	return chain.txs[chain.index]
}

// proposalHeap orders executable sender heads for interleaving across senders.
// See [proposalHeap.Less].
type proposalHeap []*proposalChain

func (h proposalHeap) Len() int { return len(h) }

// Less orders heads by higher [evm.MempoolTx.Priority], then lower
// [evm.MempoolTx.ArrivalID], then sender address, then transaction nonce.
func (h proposalHeap) Less(i, j int) bool {
	a, b := h[i].current(), h[j].current()
	if a.Priority != b.Priority {
		return a.Priority > b.Priority
	}
	if a.ArrivalID != b.ArrivalID {
		return a.ArrivalID < b.ArrivalID
	}
	if senderOrder := bytes.Compare(a.Sender[:], b.Sender[:]); senderOrder != 0 {
		return senderOrder < 0
	}
	return a.Nonce < b.Nonce
}

func (h proposalHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *proposalHeap) Push(value any) {
	*h = append(*h, value.(*proposalChain))
}

func (h *proposalHeap) Pop() any {
	old := *h
	last := len(old) - 1
	value := old[last]
	old[last] = nil
	*h = old[:last]
	return value
}

// priorityLanesMaxBytesDivisor sets the byte cap shared by the oracle and
// zero-gas lanes to 1/4 of RequestPrepareProposal.MaxTxBytes. Price writes are
// small, so a quarter of the block leaves ample room for every price write
// and routine keeper action while guaranteeing the EVM and default lanes at
// least 3/4 of the block, even if an allowlisted key floods the mempool with
// zero-fee transactions. The cap is proposer-local and not a consensus rule.
const priorityLanesMaxBytesDivisor = 4

// NewEVMPrepareProposalHandler returns Nibiru's EVM-aware proposal builder.
//
// The handler snapshots [evm.Mempool] via [evm.Mempool.Snapshot] and selects
// each sender's complete state nonce chain (contiguous live slots from the
// committed state nonce) through [executableProposalChains], prioritizes
// executable sender heads with [proposalHeap], and returns original outer bytes
// from [evm.MempoolTx.TxBytes].
//
// The proposal is built in [proposalLane] order, as assigned by
// [classifyProposalTx]:
//
//  1. Oracle lane: zero-gas actor transactions (x/sudo ZeroGasActors) that
//     call the Sai oracle contract, signed by their sender alone
//     ([isOracleLaneTx]). Any oracle execute qualifies: price writes,
//     expiration updates, and permission or admin calls.
//  2. Zero-gas lane: other zero-gas actor transactions, such as keeper calls
//     and oracle calls with a co-signer or a distinct fee payer.
//  3. EVM lane: the [proposalHeap] selection above.
//  4. Default lane: all other non-EVM transactions.
//
// The Sai oracle address is read once per proposal from committed state,
// through the x-oracle adapter named by the EVM params Wasm plugins
// ([resolveOracleLaneRule]). When the plugin is missing, the adapter is in
// fixture mode, or the query fails, the oracle lane falls back to price
// writes matched by message key ([isOraclePriceWriteMsg]).
//
// Lanes 1, 2, and 4 come from RequestPrepareProposal.Txs and keep their
// CometBFT-provided relative order. EVM transactions present in req.Txs are
// ignored so proposal construction does not rebuild the EVM index from Comet
// candidates.
//
// Putting oracle calls first means that, in a block that contains a price
// write, every later transaction, including keeper actions and trades
// submitted as EVM transactions, executes against that write rather than the
// previous block's price. This holds only when all of the following are true:
//
//   - The block contains a price write at all.
//   - CometBFT reaped the write into req.Txs. Reaping is FIFO under the block
//     byte limit, and EVM transactions also occupy the CometBFT mempool, so
//     under a large backlog the write may be absent while the EVM lane still
//     fills from the app mempool.
//   - The proposer runs this handler. ProcessProposal does not check lane
//     order, so another proposer may order the block differently.
//
// Closing the reap gap and enforcing lane order in ProcessProposal are scope
// for the lane-enforcement follow-up; this handler is proposer-local.
//
// A signer's non-EVM transactions are never reordered across lanes. Earlier
// candidates are promoted to the lane of a later candidate sharing a signer,
// so a fast-lane transaction keeps its lane (see [splitNonEVMProposalTxs]).
// Promotion is computed over all candidates before selection, so a promoted
// transaction keeps its lane even when the transaction that promoted it is
// skipped; the block stays valid. This does not track the EVM mempool, so if one account has an EVM transaction at nonce N and a zero-gas
// Cosmos transaction at sequence N+1, the Cosmos transaction is verified first,
// fails its sequence check, and waits for a later block. Zero-gas actor keys do
// not send EVM transactions in practice, so this case is accepted.
//
// Every lane shares one byte and gas budget, filled in lane order; a candidate
// that does not fit is skipped and stays in the CometBFT mempool. In addition,
// lanes 1 and 2 together may use at most 1/[priorityLanesMaxBytesDivisor] of
// RequestPrepareProposal.MaxTxBytes, filled oracle lane first. The cap bounds
// what a compromised or runaway zero-gas actor key, which pays no fees, can
// take from the EVM and default lanes.
//
// Gas is charged at each transaction's declared limit, because the default
// ProcessProposal sums declared gas against max_gas. Zero-gas actor
// transactions execute on a fixed zero gas meter but may declare a large
// limit, so with a finite max_gas, lanes 1 and 2 can use the gas budget before
// any EVM transaction is considered. A per-lane gas cap is left for the
// lane-enforcement follow-up. With max_gas = -1 the hazard does not apply.
//
// A recovered panic yields an empty proposal rather than an unfiltered EVM
// candidate set. Node-local mempool membership affects only this node's
// candidate selection; ProcessProposal and delivery validate from proposal
// bytes and application state alone.
func NewEVMPrepareProposalHandler(
	app *NibiruApp,
	verifier prepareProposalTxVerifier,
) sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) (resp abci.ResponsePrepareProposal) {
		// A handler failure must not fall back to an unfiltered EVM candidate set.
		defer func() {
			if recovered := recover(); recovered != nil {
				ctx.Logger().Error("panic recovered in EVM PrepareProposal", "panic", recovered)
				resp = abci.ResponsePrepareProposal{Txs: nil}
			}
		}()

		maxBytes := uint64(math.MaxUint64)
		if req.MaxTxBytes >= 0 {
			maxBytes = uint64(req.MaxTxBytes)
		}
		maxGas := uint64(math.MaxUint64)
		if consensusParams := ctx.ConsensusParams(); consensusParams != nil {
			if block := consensusParams.Block; block != nil && block.MaxGas >= 0 {
				maxGas = uint64(block.MaxGas)
			}
		}

		selected := make([][]byte, 0)
		var usedBytes, usedGas uint64
		// selectNonEVM charges declared gas, not executed gas, to match
		// ProcessProposal; see the max_gas note on this handler.
		// It reports whether the candidate was selected.
		selectNonEVM := func(candidate nonEVMProposalTx) bool {
			if !proposalCapacityAllows(usedBytes, proposalTxSize(candidate.txBytes), maxBytes) ||
				!proposalCapacityAllows(usedGas, candidate.gasWanted, maxGas) {
				return false
			}
			if err := verifier.PrepareProposalVerifyTxBytes(candidate.txBytes); err != nil {
				return false
			}
			selected = append(selected, candidate.txBytes)
			usedBytes += proposalTxSize(candidate.txBytes)
			usedGas += candidate.gasWanted
			return true
		}

		oracle, err := resolveOracleLaneRule(ctx, app.EvmKeeper, app.WasmKeeper)
		if err != nil && !errors.Is(err, errOracleAdapterNotConfigured) {
			ctx.Logger().Error(
				"oracle lane falls back to price writes by message key", "err", err)
		}
		lanes := splitNonEVMProposalTxs(
			app.txConfig.TxDecoder(), req.Txs, app.SudoKeeper.GetZeroGasActors(ctx),
			oracle,
		)
		// Oracle and zero-gas lanes share the priority byte cap.
		priorityMaxBytes := maxBytes / priorityLanesMaxBytesDivisor
		var priorityBytes uint64
		for _, lane := range []proposalLane{proposalLaneOracle, proposalLaneZeroGas} {
			for _, candidate := range lanes[lane] {
				size := proposalTxSize(candidate.txBytes)
				if !proposalCapacityAllows(priorityBytes, size, priorityMaxBytes) {
					continue
				}
				if selectNonEVM(candidate) {
					priorityBytes += size
				}
			}
		}

		// EVM lane: executable sender heads from the app EVM mempool.
		chains := executableProposalChains(ctx, app.EvmKeeper, app.EvmMempool.Snapshot())
		queue := proposalHeap(chains)
		heap.Init(&queue)

		for queue.Len() > 0 {
			chain := heap.Pop(&queue).(*proposalChain)
			tx := chain.current()
			if !proposalCapacityAllows(usedBytes, proposalTxSize(tx.TxBytes), maxBytes) ||
				!proposalCapacityAllows(usedGas, tx.GasWanted, maxGas) {
				// Do not expose a later nonce from this sender when its current
				// executable head does not fit.
				continue
			}
			if err := verifier.PrepareProposalVerifyTxBytes(tx.TxBytes); err != nil {
				_ = app.EvmMempool.RemoveByTxKey(tx.TxKey)
				continue
			}
			selected = append(selected, tx.TxBytes)
			usedBytes += proposalTxSize(tx.TxBytes)
			usedGas += tx.GasWanted

			chain.index++
			if chain.index < len(chain.txs) {
				heap.Push(&queue, chain)
			}
		}

		for _, candidate := range lanes[proposalLaneDefault] {
			_ = selectNonEVM(candidate)
		}

		return abci.ResponsePrepareProposal{Txs: selected}
	}
}

// executableProposalChains builds each sender's state nonce chain from an
// [evm.Mempool.Snapshot]. The committed state nonce is the account nonce when
// the account exists, and 0 when [evmstate.Keeper.GetAccount] returns nil, so
// first-time and zero-gas senders remain proposable before ante creates the
// account on deliver. A sender is included only when it owns a live slot at
// that state nonce; the chain extends through consecutive nonces and stops at
// the first gap.
func executableProposalChains(
	ctx sdk.Context,
	keeper *evmstate.Keeper,
	snapshot []evm.MempoolSender,
) []*proposalChain {
	chains := make([]*proposalChain, 0, len(snapshot))
	for _, sender := range snapshot {
		stateNonce := uint64(0)
		if account := keeper.GetAccount(ctx, sender.Sender); account != nil {
			stateNonce = account.Nonce
		}
		start := sort.Search(len(sender.Txs), func(index int) bool {
			return sender.Txs[index].Nonce >= stateNonce
		})
		if start >= len(sender.Txs) || sender.Txs[start].Nonce != stateNonce {
			continue
		}
		end := start + 1
		for end < len(sender.Txs) && sender.Txs[end].Nonce == sender.Txs[end-1].Nonce+1 {
			end++
		}
		chains = append(chains, &proposalChain{txs: sender.Txs[start:end]})
	}
	return chains
}

// proposalCapacityAllows reports whether next units fit in the remaining
// capacity without overflowing used or maximum.
func proposalCapacityAllows(used, next, maximum uint64) bool {
	return used <= maximum && next <= maximum-used
}

// proposalTxSize returns the bytes txBytes adds to a proposal as CometBFT
// counts them against RequestPrepareProposal.MaxTxBytes: the protobuf-encoded
// size inside block data, which exceeds the raw length by the field overhead.
func proposalTxSize(txBytes []byte) uint64 {
	return uint64(cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{txBytes}))
}
