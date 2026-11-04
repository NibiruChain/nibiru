package app

import (
	"errors"
	"fmt"
	"math"

	"github.com/armon/go-metrics"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"

	srvconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/telemetry"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// LaneRulesUpgradeName is the software upgrade after which ProcessProposal
// enforces the block lane rules ([verifyProposalLaneRules]). Before that
// upgrade has run, ProcessProposal behaves exactly like the stock handler.
// It is the name of [upgrades.Upgrade2_23_0], which is registered in
// [upgrades.AllUpgrades], so the binary that runs the upgrade at its plan
// height is the one that starts enforcing at the next height.
var LaneRulesUpgradeName = upgrades.Upgrade2_23_0.UpgradeName

// laneRulesActiveAt reports whether the lane rules apply to a proposal at
// height, given the height at which [LaneRulesUpgradeName] ran (0 when it has
// not run). ProcessProposal at the upgrade height itself reads pre-upgrade
// state, so enforcement starts at doneHeight+1.
func laneRulesActiveAt(doneHeight, height int64) bool {
	return doneHeight > 0 && height > doneHeight
}

// laneRulesActive reports whether this node applies the lane rules to a
// proposal at height, under its app.toml "nibiru-bft.lane-rules" policy:
// "auto" follows [laneRulesActiveAt] for [LaneRulesUpgradeName], "on"
// always applies them, and "off" never does. PrepareProposal and
// ProcessProposal both use it, so a node's proposals follow the rules it
// votes with.
func (app *NibiruApp) laneRulesActive(ctx sdk.Context, height int64) bool {
	switch app.laneRulesMode {
	case srvconfig.LaneRulesOn:
		return true
	case srvconfig.LaneRulesOff:
		return false
	default:
		doneHeight := app.UpgradeKeeper.GetDoneHeight(ctx, LaneRulesUpgradeName)
		return laneRulesActiveAt(doneHeight, height)
	}
}

// SetLaneRulesMode overrides the app.toml "nibiru-bft.lane-rules" policy read
// at construction. It is meant for tests and must not be called while the
// node handles proposals.
func (app *NibiruApp) SetLaneRulesMode(mode srvconfig.LaneRulesMode) {
	app.laneRulesMode = mode
}

// Lane rule violation classes, wrapped by the errors of
// [verifyProposalLaneRules] and used as the "reason" telemetry label.
var (
	errLaneRuleDecode      = errors.New("undecodable or unclassifiable tx")
	errLaneRuleOrder       = errors.New("oracle price writes not first")
	errLaneRulePriorityCap = errors.New("priority lane bytes over cap")
	errLaneRulePanic       = errors.New("panic in lane rule check")
)

// laneRuleReason returns the telemetry label for a lane rule error.
func laneRuleReason(err error) string {
	switch {
	case errors.Is(err, errLaneRuleDecode):
		return "decode"
	case errors.Is(err, errLaneRuleOrder):
		return "order"
	case errors.Is(err, errLaneRulePriorityCap):
		return "priority_cap"
	case errors.Is(err, errLaneRulePanic):
		return "panic"
	default:
		return "other"
	}
}

// Telemetry keys for the lane rules: rejected proposals in ProcessProposal
// and PrepareProposal self-check fallbacks to an empty proposal.
var (
	laneRulesRejectMetric   = []string{"nibiru_bft", "lane_rules", "reject"}
	laneRulesFallbackMetric = []string{"nibiru_bft", "lane_rules", "prepare_fallback"}
)

// countLaneRuleError increments metric with the reason label of err.
func countLaneRuleError(metric []string, err error) {
	telemetry.IncrCounterWithLabels(metric, 1,
		[]metrics.Label{telemetry.NewLabel("reason", laneRuleReason(err))})
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
			return nil, nil, fmt.Errorf(
				"tx %d: %w: %w", idx, errLaneRuleDecode, err)
		}
		lane, signers, ok := classifyProposalCandidate(tx, zeroGasActors)
		if !ok {
			return nil, nil, fmt.Errorf("tx %d: %w", idx, errLaneRuleDecode)
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

// choosePriorityProposalTxs picks the oracle and zero-gas candidates that
// PrepareProposal will try to propose while the lane rules are active,
// before any of them is verified. It walks lanes[proposalLaneOracle] then
// lanes[proposalLaneZeroGas] from [splitNonEVMProposalTxs] and keeps each
// candidate that fits the priority byte cap, the block byte limit maxBytes,
// and the gas limit maxGas. A candidate that later fails verification still
// counts against these budgets for this proposal; the EVM and default lanes
// then get the space it would have used.
func choosePriorityProposalTxs(
	lanes [proposalLaneCount][]nonEVMProposalTx,
	priorityMaxBytes, maxBytes, maxGas uint64,
) []nonEVMProposalTx {
	var chosen []nonEVMProposalTx
	var priorityBytes, usedGas uint64
	for _, lane := range []proposalLane{proposalLaneOracle, proposalLaneZeroGas} {
		for _, candidate := range lanes[lane] {
			size := proposalTxSize(candidate.txBytes)
			if !proposalCapacityAllows(priorityBytes, size, priorityMaxBytes) ||
				!proposalCapacityAllows(priorityBytes, size, maxBytes) ||
				!proposalCapacityAllows(usedGas, candidate.gasWanted, maxGas) {
				continue
			}
			chosen = append(chosen, candidate)
			priorityBytes += size
			usedGas += candidate.gasWanted
		}
	}
	return chosen
}

// priorityProposalGroup is a run of priority transactions that
// [verifyPriorityProposalTxs] verifies and proposes in order. A guarded
// group is an oracle lane component holding promoted transactions as well
// as price writes, whose price writes are proposed only when that keeps the
// lane rules satisfiable.
type priorityProposalGroup struct {
	txs     []nonEVMProposalTx
	guarded bool
}

// orderPriorityProposalTxs fixes the order in which PrepareProposal
// verifies and proposes chosen, the result of [choosePriorityProposalTxs].
// Ante has effects outside the signers' accounts, so the order in which
// PrepareProposal verifies transactions must be the order it proposes them;
// otherwise ProcessProposal, which replays ante in block order, can reject a
// block that PrepareProposal accepted.
//
// Promotion is recomputed over chosen alone with [promoteProposalLanes],
// because the budgets skip candidates: a transaction promoted for a price
// write that was not chosen no longer precedes the price writes. The groups
// are, in order:
//
//  1. Oracle lane components, grouped by shared signers, whose every member
//     is a price write. Such a group never makes another transaction break
//     the oracle-first rule.
//  2. The other oracle lane components, guarded: each also holds
//     transactions promoted ahead of a later price write of the component.
//  3. The zero-gas lane.
//
// Components share no signers within the oracle lane, and each keeps its
// candidate order, so no signer's transactions are reordered, and promotion
// over the result equals promotion over chosen. The order is computed before
// any transaction is verified.
func orderPriorityProposalTxs(
	chosen []nonEVMProposalTx,
) []priorityProposalGroup {
	entries := make([]proposalLaneEntry, len(chosen))
	for idx, candidate := range chosen {
		entries[idx] = candidate.entry
	}
	promoteProposalLanes(entries)

	// Union the oracle lane by shared signer. component maps a signer to the
	// index of the first oracle lane tx in its component; parents and the
	// path halving in find keep the union-find near linear.
	parents := make([]int, len(chosen))
	find := func(idx int) int {
		for parents[idx] != idx {
			parents[idx] = parents[parents[idx]]
			idx = parents[idx]
		}
		return idx
	}
	firstBySigner := make(map[string]int)
	for idx, entry := range entries {
		parents[idx] = idx
		if entry.lane != proposalLaneOracle {
			continue
		}
		for _, signer := range entry.signers {
			if first, seen := firstBySigner[signer]; seen {
				root, other := find(first), find(idx)
				// Keep the earliest index as the root so components keep
				// their order of first appearance.
				parents[max(root, other)] = min(root, other)
			} else {
				firstBySigner[signer] = idx
			}
		}
	}

	componentOf := make(map[int]int) // root index -> component position
	var components []priorityProposalGroup
	var zeroGas priorityProposalGroup
	for idx, entry := range entries {
		if entry.lane != proposalLaneOracle {
			zeroGas.txs = append(zeroGas.txs, chosen[idx])
			continue
		}
		root := find(idx)
		pos, seen := componentOf[root]
		if !seen {
			pos = len(components)
			componentOf[root] = pos
			components = append(components, priorityProposalGroup{})
		}
		components[pos].txs = append(components[pos].txs, chosen[idx])
		if chosen[idx].entry.lane != proposalLaneOracle {
			components[pos].guarded = true
		}
	}

	groups := make([]priorityProposalGroup, 0, len(components)+1)
	for _, guarded := range []bool{false, true} {
		for _, component := range components {
			if component.guarded == guarded {
				groups = append(groups, component)
			}
		}
	}
	return append(groups, zeroGas)
}

// verifyPriorityProposalTxs verifies groups from [orderPriorityProposalTxs]
// in order and returns the transactions that passed, in that same order,
// which is the order PrepareProposal proposes them.
//
// A verification failure can strand a promoted transaction: feeder A's call
// X was verified and proposed ahead of A's price write, and then the price
// write failed. X now has no later price write to justify it, so no price
// write may follow it in the block. Since X cannot be withdrawn without
// changing the state the later transactions were verified against, a price
// write in a guarded group is proposed, and verified, only when every
// transaction already proposed in its group is justified by the proposed
// transactions of the group plus that price write, and no earlier guarded
// group left a stranded transaction. A skipped price write stays in the
// mempool for a later block. Price-write-only groups come first and cannot
// strand anything, so a failure in a guarded group never costs them.
//
// The rule in ProcessProposal is not relaxed for this case instead, for
// example by admitting before a price write any transaction whose signers
// are all allowlisted senders, because that would let a proposer place
// keeper and admin transactions ahead of the block's price writes, which is
// what the rule exists to prevent.
//
// Every transaction proposed before the last proposed price write is then
// justified by proposed transactions, and later transactions only add
// promotions, so the block satisfies [verifyProposalLaneRules].
func verifyPriorityProposalTxs(
	groups []priorityProposalGroup,
	verifier prepareProposalTxVerifier,
) []nonEVMProposalTx {
	var proposed []nonEVMProposalTx
	stranded := false
	for _, group := range groups {
		var groupEntries []proposalLaneEntry
		for _, candidate := range group.txs {
			if group.guarded && candidate.entry.lane == proposalLaneOracle &&
				(stranded || !allProposalEntriesOracle(
					append(groupEntries, candidate.entry))) {
				continue
			}
			if err := verifier.PrepareProposalVerifyTxBytes(candidate.txBytes); err != nil {
				continue
			}
			proposed = append(proposed, candidate)
			if group.guarded {
				groupEntries = append(groupEntries, candidate.entry)
			}
		}
		if group.guarded && !allProposalEntriesOracle(groupEntries) {
			stranded = true
		}
	}
	return proposed
}

// allProposalEntriesOracle reports whether [promoteProposalLanes] over
// entries, in order, puts every entry in [proposalLaneOracle]. entries is
// not modified.
func allProposalEntriesOracle(entries []proposalLaneEntry) bool {
	promoted := make([]proposalLaneEntry, len(entries))
	copy(promoted, entries)
	promoteProposalLanes(promoted)
	for _, entry := range promoted {
		if entry.lane != proposalLaneOracle {
			return false
		}
	}
	return true
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
				"%w: tx %d in %s lane precedes oracle price write tx %d",
				errLaneRuleOrder, idx, promoted[idx], lastPriceWrite,
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
				"%w: tx %d exceeds cap %d", errLaneRulePriorityCap, idx,
				priorityMaxBytes,
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
			err = fmt.Errorf("%w: %v", errLaneRulePanic, recovered)
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
// While [NibiruApp.laneRulesActive] is false, it calls next unchanged. Under
// the default "auto" policy, that is until [LaneRulesUpgradeName] has run
// and at its own height. Afterward, it rejects a proposal that has an
// undecodable or unclassifiable transaction or that breaks a lane rule, and
// otherwise calls next. Classification uses only the proposal bytes and
// committed state: the x/sudo ZeroGasActors and consensus params that
// PrepareProposal also reads. No client tags or node-local mempool state are
// involved. Each rejection is logged at Error level and counted in the
// nibiru_bft.lane_rules.reject telemetry counter, labeled by reason.
func NewLaneOrderProcessProposalHandler(
	app *NibiruApp,
	next sdk.ProcessProposalHandler,
) sdk.ProcessProposalHandler {
	return func(
		ctx sdk.Context,
		req abci.RequestProcessProposal,
	) abci.ResponseProcessProposal {
		if !app.laneRulesActive(ctx, req.Height) {
			return next(ctx, req)
		}
		if err := checkProposalLaneRules(ctx, app, req.Txs); err != nil {
			countLaneRuleError(laneRulesRejectMetric, err)
			ctx.Logger().Error(
				"rejecting proposal: block lane rules",
				"height", req.Height, "reason", laneRuleReason(err), "err", err,
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
