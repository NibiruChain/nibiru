package app

import (
	"bytes"
	"container/heap"
	"math"
	"sort"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

type prepareProposalTxVerifier interface {
	PrepareProposalVerifyTxBytes(txBytes []byte) error
}

type proposalChain struct {
	txs   []evm.MempoolTx
	index int
}

func (chain *proposalChain) current() evm.MempoolTx {
	return chain.txs[chain.index]
}

type proposalHeap []*proposalChain

func (h proposalHeap) Len() int { return len(h) }

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

// NewEVMPrepareProposalHandler returns Nibiru's EVM-aware proposal builder. It
// selects complete state nonce chains from the node-local EVM mempool before
// appending valid non-EVM candidates in their CometBFT-provided order.
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
		chains := executableProposalChains(ctx, app.EvmKeeper, app.EvmMempool.Snapshot())
		queue := proposalHeap(chains)
		heap.Init(&queue)

		for queue.Len() > 0 {
			chain := heap.Pop(&queue).(*proposalChain)
			tx := chain.current()
			if !proposalCapacityAllows(usedBytes, uint64(len(tx.TxBytes)), maxBytes) ||
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
			usedBytes += uint64(len(tx.TxBytes))
			usedGas += tx.GasWanted

			chain.index++
			if chain.index < len(chain.txs) {
				heap.Push(&queue, chain)
			}
		}

		for _, txBytes := range req.Txs {
			tx, err := app.txConfig.TxDecoder()(txBytes)
			if err != nil || evm.IsEthTx(tx) {
				continue
			}
			gasWanted := uint64(0)
			if gasTx, ok := tx.(interface{ GetGas() uint64 }); ok {
				gasWanted = gasTx.GetGas()
			}
			if !proposalCapacityAllows(usedBytes, uint64(len(txBytes)), maxBytes) ||
				!proposalCapacityAllows(usedGas, gasWanted, maxGas) {
				continue
			}
			if err := verifier.PrepareProposalVerifyTxBytes(txBytes); err != nil {
				continue
			}
			selected = append(selected, txBytes)
			usedBytes += uint64(len(txBytes))
			usedGas += gasWanted
		}

		return abci.ResponsePrepareProposal{Txs: selected}
	}
}

func executableProposalChains(
	ctx sdk.Context,
	keeper *evmstate.Keeper,
	snapshot []evm.MempoolSender,
) []*proposalChain {
	chains := make([]*proposalChain, 0, len(snapshot))
	for _, sender := range snapshot {
		account := keeper.GetAccount(ctx, sender.Sender)
		if account == nil {
			continue
		}
		start := sort.Search(len(sender.Txs), func(index int) bool {
			return sender.Txs[index].Nonce >= account.Nonce
		})
		if start >= len(sender.Txs) || sender.Txs[start].Nonce != account.Nonce {
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

func proposalCapacityAllows(used, next, maximum uint64) bool {
	return used <= maximum && next <= maximum-used
}
