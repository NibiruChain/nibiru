package gov

import (
	"fmt"
	"time"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/baseapp"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/telemetry"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/keeper"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	v1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"
)

// EndBlocker called every block, process inflation, update validator set.
func EndBlocker(ctx sdk.Context, keeper *keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	logger := keeper.Logger(ctx)

	// fail queued proposals that can't be unmarshaled before the loops below
	// load them, since GetProposal panics on them and would halt the chain.
	// This can happen after an upgrade removes a message type that a queued
	// proposal uses.
	inactive := collectUndecodableProposals(ctx, keeper,
		keeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time),
		types.SplitInactiveProposalQueueKey)
	for _, p := range inactive {
		failUndecodableProposal(ctx, keeper, p, false)
	}

	// delete dead proposals from store and returns theirs deposits.
	// A proposal is dead when it's inactive and didn't get enough deposit on time to get into voting phase.
	keeper.IterateInactiveProposalsQueue(ctx, ctx.BlockHeader().Time, func(proposal v1.Proposal) bool {
		keeper.DeleteProposal(ctx, proposal.Id)

		params := keeper.GetParams(ctx)
		if !params.BurnProposalDepositPrevote {
			keeper.RefundAndDeleteDeposits(ctx, proposal.Id) // refund deposit if proposal got removed without getting 100% of the proposal
		} else {
			keeper.DeleteAndBurnDeposits(ctx, proposal.Id) // burn the deposit if proposal got removed without getting 100% of the proposal
		}

		// called when proposal become inactive
		keeper.Hooks().AfterProposalFailedMinDeposit(ctx, proposal.Id)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeInactiveProposal,
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
				sdk.NewAttribute(types.AttributeKeyProposalResult, types.AttributeValueProposalDropped),
			),
		)

		logger.Info(
			"proposal did not meet minimum deposit; deleted",
			"proposal", proposal.Id,
			"min_deposit", sdk.NewCoins(params.MinDeposit...).String(),
			"total_deposit", sdk.NewCoins(proposal.TotalDeposit...).String(),
		)

		return false
	})

	active := collectUndecodableProposals(ctx, keeper,
		keeper.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time),
		types.SplitActiveProposalQueueKey)
	for _, p := range active {
		failUndecodableProposal(ctx, keeper, p, true)
	}

	// fetch active proposals whose voting periods have ended (are passed the block time)
	keeper.IterateActiveProposalsQueue(ctx, ctx.BlockHeader().Time, func(proposal v1.Proposal) bool {
		var tagValue, logMsg string

		passes, burnDeposits, tallyResults := keeper.Tally(ctx, proposal)

		if burnDeposits {
			keeper.DeleteAndBurnDeposits(ctx, proposal.Id)
		} else {
			keeper.RefundAndDeleteDeposits(ctx, proposal.Id)
		}

		if passes {
			var (
				idx    int
				events sdk.Events
				msg    sdk.Msg
			)

			// attempt to execute all messages within the passed proposal
			// Messages may mutate state thus we use a cached context. If one of
			// the handlers fails, no state mutation is written and the error
			// message is logged.
			cacheCtx, writeCache := ctx.CacheContext()
			messages, err := proposal.GetMsgs()
			if err == nil {
				for idx, msg = range messages {
					handler := keeper.Router().Handler(msg)
					var res *sdk.Result
					res, err = safeExecuteHandler(cacheCtx, msg, handler)
					if err != nil {
						break
					}

					events = append(events, res.GetEvents()...)
				}
			}

			// `err == nil` when all handlers passed.
			// Or else, `idx` and `err` are populated with the msg index and error.
			if err == nil {
				proposal.Status = v1.StatusPassed
				tagValue = types.AttributeValueProposalPassed
				logMsg = "passed"

				// write state to the underlying multi-store
				writeCache()

				// propagate the msg events to the current context
				ctx.EventManager().EmitEvents(events)
			} else {
				proposal.Status = v1.StatusFailed
				tagValue = types.AttributeValueProposalFailed
				logMsg = fmt.Sprintf("passed, but msg %d (%s) failed on execution: %s", idx, sdk.MsgTypeURL(msg), err)
			}
		} else {
			proposal.Status = v1.StatusRejected
			tagValue = types.AttributeValueProposalRejected
			logMsg = "rejected"
		}

		proposal.FinalTallyResult = &tallyResults

		keeper.SetProposal(ctx, proposal)
		keeper.RemoveFromActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)

		// when proposal become active
		keeper.Hooks().AfterProposalVotingPeriodEnded(ctx, proposal.Id)

		logger.Info(
			"proposal tallied",
			"proposal", proposal.Id,
			"results", logMsg,
		)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeActiveProposal,
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
				sdk.NewAttribute(types.AttributeKeyProposalResult, tagValue),
			),
		)
		return false
	})
}

// executes handle(msg) and recovers from panic.
func safeExecuteHandler(ctx sdk.Context, msg sdk.Msg, handler baseapp.MsgServiceHandler,
) (res *sdk.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handling x/gov proposal msg [%s] PANICKED: %v", msg, r)
		}
	}()
	res, err = handler(ctx, msg)
	return
}

// undecodableProposal is a queued proposal that x/gov can't unmarshal.
type undecodableProposal struct {
	proposal v1.Proposal
	endTime  time.Time
	err      error
}

// collectUndecodableProposals returns the proposals in a proposal queue
// iterator that can't be unmarshaled. It reads the whole queue range before
// the caller fails any of them, so the store isn't written while the iterator
// is open.
func collectUndecodableProposals(
	ctx sdk.Context,
	keeper *keeper.Keeper,
	iterator sdk.Iterator,
	splitKey func(key []byte) (proposalID uint64, endTime time.Time),
) []undecodableProposal {
	defer iterator.Close()

	var out []undecodableProposal
	for ; iterator.Valid(); iterator.Next() {
		proposalID, endTime := splitKey(iterator.Key())
		proposal, found, err := keeper.GetProposalE(ctx, proposalID)
		if found && err != nil {
			out = append(out, undecodableProposal{
				proposal: proposal,
				endTime:  endTime,
				err:      err,
			})
		}
	}
	return out
}

// failUndecodableProposal refunds deposits and removes the queue entry. It
// deletes inactive proposals and retains failed records for active proposals.
// It skips gov hooks because they may load the proposal.
func failUndecodableProposal(
	ctx sdk.Context,
	keeper *keeper.Keeper,
	p undecodableProposal,
	active bool,
) {
	proposal := p.proposal
	eventType := types.EventTypeInactiveProposal
	if active {
		eventType = types.EventTypeActiveProposal
		keeper.FailVotingProposal(ctx, proposal, p.endTime)
	} else {
		proposal.Status = v1.StatusFailed
		proposal.Messages = nil
		keeper.SetProposal(ctx, proposal)
		keeper.RefundAndDeleteDeposits(ctx, proposal.Id)
		keeper.RemoveFromInactiveProposalQueue(ctx, proposal.Id, p.endTime)
		keeper.DeleteProposal(ctx, proposal.Id)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
			sdk.NewAttribute(types.AttributeKeyProposalResult, types.AttributeValueProposalFailed),
		),
	)

	keeper.Logger(ctx).Info(
		"proposal failed because it cannot be processed by gov",
		"proposal", proposal.Id,
		"active", active,
		"title", proposal.Title,
		"error", p.err.Error(),
	)
}
