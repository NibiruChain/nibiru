package app

import (
	"fmt"
	"math"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// LaneRulesUpgradeName is the software upgrade after which ProcessProposal
// enforces the block lane rules ([verifyProposalLaneRules]). Before that
// upgrade has run, ProcessProposal behaves exactly like the stock handler.
//
// TODO: Tie this to the v2.23.0 release. The upgrade itself needs no state
// migration for the lane rules, but its plan name must match this constant.
const LaneRulesUpgradeName = "v2.23.0"

// laneRulesActiveAt reports whether the lane rules apply to a proposal at
// height, given the height at which [LaneRulesUpgradeName] ran (0 when it has
// not run). ProcessProposal at the upgrade height itself reads pre-upgrade
// state, so enforcement starts at doneHeight+1.
func laneRulesActiveAt(doneHeight, height int64) bool {
	return doneHeight > 0 && height > doneHeight
}

// priorityLanesMaxBytes returns the consensus byte cap on transactions in
// [proposalLaneOracle] and [proposalLaneZeroGas]: the committed consensus
// param Block.MaxBytes divided by [priorityLanesMaxBytesDivisor]. It returns
// math.MaxUint64, meaning no cap, when the params are unavailable.
//
// The cap is defined on Block.MaxBytes rather than on
// RequestPrepareProposal.MaxTxBytes because ProcessProposal cannot recompute
// MaxTxBytes: CometBFT derives it from Block.MaxBytes minus header, commit,
// and evidence overhead, which depend on the validator set and the proposer's
// pending evidence. Both handlers read Block.MaxBytes from the same committed
// state, so they agree exactly. PrepareProposal applies the tighter of this
// cap and MaxTxBytes / [priorityLanesMaxBytesDivisor]; since MaxTxBytes is
// below Block.MaxBytes, an honest proposal always fits this cap.
func priorityLanesMaxBytes(ctx sdk.Context) uint64 {
	consensusParams := ctx.ConsensusParams()
	if consensusParams == nil || consensusParams.Block == nil ||
		consensusParams.Block.MaxBytes <= 0 {
		return math.MaxUint64
	}
	return uint64(consensusParams.Block.MaxBytes) / priorityLanesMaxBytesDivisor
}

// proposalBlockLanes classifies txs in the given order and applies
// [promoteProposalLanes] over them, as if they were the whole candidate list.
// It returns each transaction's own lane from [classifyProposalTx] and its
// promoted lane. EVM transactions keep [proposalLaneEVM] in both.
//
// An undecodable transaction, or one whose classification panics, is an
// error. PrepareProposal never selects either, so only a faulty or malicious
// proposer produces them.
func proposalBlockLanes(
	decode sdk.TxDecoder,
	txs [][]byte,
	zeroGasActors sudo.ZeroGasActors,
) (own []proposalLane, promoted []proposalLane, err error) {
	entries := make([]proposalLaneEntry, len(txs))
	own = make([]proposalLane, len(txs))
	for idx, txBytes := range txs {
		tx, err := decode(txBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("tx %d: undecodable: %w", idx, err)
		}
		lane, signers, ok := classifyProposalCandidate(tx, zeroGasActors)
		if !ok {
			return nil, nil, fmt.Errorf("tx %d: unclassifiable", idx)
		}
		own[idx] = lane
		entries[idx] = newProposalLaneEntry(lane, signers)
	}
	promoteProposalLanes(entries)
	promoted = make([]proposalLane, len(txs))
	for idx, entry := range entries {
		promoted[idx] = entry.lane
	}
	return own, promoted, nil
}

// orderPriorityProposalTxs reorders the transactions PrepareProposal selected
// for [proposalLaneOracle] and [proposalLaneZeroGas] so the block satisfies
// [verifyProposalLaneRules]. It recomputes promotion over the selected
// transactions alone with [proposalBlockLanes] and groups them by the
// recomputed lane, keeping relative order within each lane.
//
// The regrouping is needed because [splitNonEVMProposalTxs] promotes over
// the full candidate list, but the block is a subsequence of it: the byte
// cap, the gas limit, and the verifier skip candidates. A transaction
// promoted into the oracle lane by a later same-signer price write whose
// price write was skipped no longer has a reason to precede the other price
// writes, and if it stayed ahead of them, every validator would reject the
// block. Regrouping moves it behind the price writes instead of dropping it.
//
// Regrouping never reorders one signer's transactions (see
// [promoteProposalLanes]), so each signer's sequence order, as verified,
// still holds. It can move a transaction relative to transactions with which
// it shares no signer. On error, which cannot happen for candidates that
// [splitNonEVMProposalTxs] already decoded and classified, txs are returned
// unchanged.
func orderPriorityProposalTxs(
	decode sdk.TxDecoder,
	txs [][]byte,
	zeroGasActors sudo.ZeroGasActors,
) [][]byte {
	_, lanes, err := proposalBlockLanes(decode, txs, zeroGasActors)
	if err != nil {
		return txs
	}
	ordered := make([][]byte, 0, len(txs))
	for lane := range proposalLaneCount {
		for idx, txBytes := range txs {
			if lanes[idx] == lane {
				ordered = append(ordered, txBytes)
			}
		}
	}
	return ordered
}

// verifyProposalLaneRules checks the block lane rules that ProcessProposal
// enforces after [LaneRulesUpgradeName]. With lanes from
// [proposalBlockLanes] over the whole block:
//
//  1. Oracle first: every transaction before the last oracle price write
//     (a transaction whose own lane is [proposalLaneOracle]) has promoted
//     lane [proposalLaneOracle]. That is, a price write may be preceded only
//     by price writes and by transactions that share a signer, directly or
//     through a chain of later transactions, with a later price write in the
//     same block. EVM transactions are never in the oracle lane, so an EVM
//     transaction before a price write breaks the rule. A block without a
//     price write satisfies it. The relative order of the zero-gas, EVM, and
//     default lanes is not checked.
//  2. Priority byte cap: the protobuf-encoded size ([proposalTxSize]) of
//     transactions whose promoted lane is [proposalLaneOracle] or
//     [proposalLaneZeroGas] sums to at most priorityMaxBytes
//     ([priorityLanesMaxBytes]).
//
// Promoting over the block rather than over the proposer's candidates keeps
// the rule a function of the block and committed state alone. Any lane a
// transaction gets from promotion over the block is at least the lane it got
// over the candidates, because the block keeps each signer's candidate order,
// so the priority bytes counted here never exceed what PrepareProposal
// counted against the same cap.
func verifyProposalLaneRules(
	decode sdk.TxDecoder,
	txs [][]byte,
	zeroGasActors sudo.ZeroGasActors,
	priorityMaxBytes uint64,
) error {
	own, promoted, err := proposalBlockLanes(decode, txs, zeroGasActors)
	if err != nil {
		return err
	}

	lastPriceWrite := -1
	for idx, lane := range own {
		if lane == proposalLaneOracle {
			lastPriceWrite = idx
		}
	}
	for idx := range lastPriceWrite {
		if promoted[idx] != proposalLaneOracle {
			return fmt.Errorf(
				"tx %d in %s lane precedes oracle price write tx %d",
				idx, promoted[idx], lastPriceWrite,
			)
		}
	}

	var priorityBytes uint64
	for idx, lane := range promoted {
		if lane != proposalLaneOracle && lane != proposalLaneZeroGas {
			continue
		}
		size := proposalTxSize(txs[idx])
		if !proposalCapacityAllows(priorityBytes, size, priorityMaxBytes) {
			return fmt.Errorf(
				"tx %d: priority lane bytes exceed cap %d", idx, priorityMaxBytes,
			)
		}
		priorityBytes += size
	}
	return nil
}

// checkProposalLaneRules runs [verifyProposalLaneRules] against the
// proposal's committed state. A panic is returned as an error so that a
// malformed block is rejected rather than crashing the node or depending on
// an outer recover.
func checkProposalLaneRules(
	ctx sdk.Context,
	app *NibiruApp,
	txs [][]byte,
) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic in lane rule check: %v", recovered)
		}
	}()
	return verifyProposalLaneRules(
		app.txConfig.TxDecoder(),
		txs,
		app.SudoKeeper.GetZeroGasActors(ctx),
		priorityLanesMaxBytes(ctx),
	)
}

// NewLaneOrderProcessProposalHandler wraps next, the stock ProcessProposal
// handler, with the block lane rules of [verifyProposalLaneRules].
//
// Until [LaneRulesUpgradeName] has run and for its own height, it calls next
// unchanged. Afterward, it rejects a proposal that has an undecodable or
// unclassifiable transaction or that breaks a lane rule, and otherwise calls
// next. Classification uses only the proposal bytes and committed state: the
// x/sudo ZeroGasActors and consensus params that PrepareProposal also reads.
// No client tags or node-local mempool state are involved.
func NewLaneOrderProcessProposalHandler(
	app *NibiruApp,
	next sdk.ProcessProposalHandler,
) sdk.ProcessProposalHandler {
	return func(
		ctx sdk.Context,
		req abci.RequestProcessProposal,
	) abci.ResponseProcessProposal {
		doneHeight := app.UpgradeKeeper.GetDoneHeight(ctx, LaneRulesUpgradeName)
		if !laneRulesActiveAt(doneHeight, req.Height) {
			return next(ctx, req)
		}
		if err := checkProposalLaneRules(ctx, app, req.Txs); err != nil {
			ctx.Logger().Info(
				"rejecting proposal: block lane rules",
				"height", req.Height, "err", err,
			)
			return abci.ResponseProcessProposal{
				Status: abci.ResponseProcessProposal_REJECT,
			}
		}
		return next(ctx, req)
	}
}

// String returns the lane name used in lane rule errors.
func (lane proposalLane) String() string {
	switch lane {
	case proposalLaneOracle:
		return "oracle"
	case proposalLaneZeroGas:
		return "zero-gas"
	case proposalLaneEVM:
		return "EVM"
	case proposalLaneDefault:
		return "default"
	default:
		return fmt.Sprintf("proposalLane(%d)", uint8(lane))
	}
}
