package app

import (
	"bytes"
	"container/heap"
	"math"
	"sort"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
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

// NewEVMPrepareProposalHandler returns Nibiru's EVM-aware proposal builder.
//
// The handler snapshots [evm.Mempool] via [evm.Mempool.Snapshot] and selects
// each sender's complete state nonce chain (contiguous live slots from the
// committed state nonce) through [executableProposalChains], prioritizes
// executable sender heads with [proposalHeap], and returns original outer bytes
// from [evm.MempoolTx.TxBytes]. Non-EVM candidates from
// RequestPrepareProposal.Txs keep their CometBFT-provided relative order, split
// into two lanes: zero-gas actor transactions ([ante.IsZeroGasActorTx], e.g.
// oracle price writes) are selected before every EVM transaction, and all other
// non-EVM transactions are appended after them. Without the leading lane, a
// trade submitted as an EVM transaction would always execute against the
// previous block's price while the oracle update for that block waits behind
// it. EVM transactions present in req.Txs are ignored so proposal construction
// does not rebuild the EVM index from Comet candidates. A recovered panic yields an empty
// proposal rather than an unfiltered EVM candidate set. Node-local mempool
// membership affects only this node's candidate selection; ProcessProposal and
// delivery validate from proposal bytes and application state alone.
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
		selectNonEVM := func(candidate nonEVMProposalTx) {
			if !proposalCapacityAllows(usedBytes, proposalTxSize(candidate.txBytes), maxBytes) ||
				!proposalCapacityAllows(usedGas, candidate.gasWanted, maxGas) {
				return
			}
			if err := verifier.PrepareProposalVerifyTxBytes(candidate.txBytes); err != nil {
				return
			}
			selected = append(selected, candidate.txBytes)
			usedBytes += proposalTxSize(candidate.txBytes)
			usedGas += candidate.gasWanted
		}

		zeroGasLane, otherLane := splitNonEVMProposalTxs(
			app.txConfig.TxDecoder(), req.Txs, app.SudoKeeper.GetZeroGasActors(ctx),
		)
		for _, candidate := range zeroGasLane {
			selectNonEVM(candidate)
		}

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

		for _, candidate := range otherLane {
			selectNonEVM(candidate)
		}

		return abci.ResponsePrepareProposal{Txs: selected}
	}
}

// nonEVMProposalTx is a decoded non-EVM candidate from
// RequestPrepareProposal.Txs with its original outer bytes.
type nonEVMProposalTx struct {
	txBytes   []byte
	gasWanted uint64
}

// splitNonEVMProposalTxs decodes the CometBFT candidates once and splits the
// non-EVM transactions into the zero-gas actor lane and the remaining lane,
// preserving relative order within each. Undecodable and EVM candidates are
// dropped.
func splitNonEVMProposalTxs(
	decode sdk.TxDecoder,
	candidates [][]byte,
	zeroGasActors sudo.ZeroGasActors,
) (zeroGasLane, otherLane []nonEVMProposalTx) {
	for _, txBytes := range candidates {
		tx, err := decode(txBytes)
		if err != nil || evm.IsEthTx(tx) {
			continue
		}
		candidate := nonEVMProposalTx{txBytes: txBytes}
		if gasTx, ok := tx.(interface{ GetGas() uint64 }); ok {
			candidate.gasWanted = gasTx.GetGas()
		}
		if ante.IsZeroGasActorTx(tx, zeroGasActors) {
			zeroGasLane = append(zeroGasLane, candidate)
		} else {
			otherLane = append(otherLane, candidate)
		}
	}
	return zeroGasLane, otherLane
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
