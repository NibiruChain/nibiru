package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/errors"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the oracle MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (ms msgServer) AggregateExchangeRatePrevote(
	goCtx context.Context,
	msg *types.MsgAggregateExchangeRatePrevote,
) (*types.MsgAggregateExchangeRatePrevoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	valAddr, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, err
	}

	feederAddr, err := sdk.AccAddressFromBech32(msg.Feeder)
	if err != nil {
		return nil, err
	}

	if err := ms.ValidateFeeder(ctx, feederAddr, valAddr); err != nil {
		return nil, err
	}

	// Convert hex string to votehash
	voteHash, err := types.AggregateVoteHashFromHexString(msg.Hash)
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrInvalidHash, err.Error())
	}

	ms.Keeper.Prevotes.Insert(ctx, valAddr, types.NewAggregateExchangeRatePrevote(voteHash, valAddr, uint64(ctx.BlockHeight())))

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAggregatePrevote,
			sdk.NewAttribute(types.AttributeKeyVoter, msg.Validator),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Feeder),
		),
	})

	return &types.MsgAggregateExchangeRatePrevoteResponse{}, nil
}

func (ms msgServer) AggregateExchangeRateVote(goCtx context.Context, msg *types.MsgAggregateExchangeRateVote) (*types.MsgAggregateExchangeRateVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	valAddr, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, err
	}

	feederAddr, err := sdk.AccAddressFromBech32(msg.Feeder)
	if err != nil {
		return nil, err
	}

	if err := ms.ValidateFeeder(ctx, feederAddr, valAddr); err != nil {
		return nil, err
	}

	params, err := ms.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	// An aggergate prevote is required to get an aggregate vote.
	aggregatePrevote, err := ms.Keeper.Prevotes.Get(ctx, valAddr)
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrNoAggregatePrevote, msg.Validator)
	}

	// Check a msg is submitted proper period
	// This condition necessary for the commit-reveal scheme.
	if (uint64(ctx.BlockHeight())/params.VotePeriod)-(aggregatePrevote.SubmitBlock/params.VotePeriod) != 1 {
		return nil, types.ErrRevealPeriodMissMatch.Wrapf(
			"aggregate prevote block: %d, current block: %d, vote period: %d",
			aggregatePrevote.SubmitBlock, ctx.BlockHeight(), params.VotePeriod,
		)
	}

	// Slice of (Pair, ExchangeRate) tuples.
	exchangeRateTuples, err := types.ParseExchangeRateTuples(msg.ExchangeRates)
	if err != nil {
		return nil, sdkerrors.Wrap(errors.ErrInvalidCoins, err.Error())
	}

	// Check all pairs are in the vote target
	for _, tuple := range exchangeRateTuples {
		if !ms.IsWhitelistedPair(ctx, tuple.Pair) {
			return nil, sdkerrors.Wrap(types.ErrUnknownPair, tuple.Pair.String())
		}
	}

	// Verify an exchange rate with aggregate prevote hash
	hash := types.GetAggregateVoteHash(msg.Salt, msg.ExchangeRates, valAddr)
	if aggregatePrevote.Hash != hash.String() {
		return nil, sdkerrors.Wrapf(types.ErrVerificationFailed, "must be given %s not %s", aggregatePrevote.Hash, hash)
	}

	// Move aggregate prevote to aggregate vote with given exchange rates
	ms.Keeper.Votes.Insert(ctx, valAddr, types.NewAggregateExchangeRateVote(exchangeRateTuples, valAddr))
	_ = ms.Keeper.Prevotes.Delete(ctx, valAddr)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAggregateVote,
			sdk.NewAttribute(types.AttributeKeyVoter, msg.Validator),
			sdk.NewAttribute(types.AttributeKeyExchangeRates, msg.ExchangeRates),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Feeder),
		),
	})

	return &types.MsgAggregateExchangeRateVoteResponse{}, nil
}

func (ms msgServer) DelegateFeedConsent(goCtx context.Context, msg *types.MsgDelegateFeedConsent) (*types.MsgDelegateFeedConsentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	operatorAddr, err := sdk.ValAddressFromBech32(msg.Operator)
	if err != nil {
		return nil, err
	}

	delegateAddr, err := sdk.AccAddressFromBech32(msg.Delegate)
	if err != nil {
		return nil, err
	}

	// Check the delegator is a validator
	val := ms.StakingKeeper.Validator(ctx, operatorAddr)
	if val == nil {
		return nil, sdkerrors.Wrap(stakingtypes.ErrNoValidatorFound, msg.Operator)
	}

	// Set the delegation
	ms.Keeper.FeederDelegations.Insert(ctx, operatorAddr, delegateAddr)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeFeedDelegate,
			sdk.NewAttribute(types.AttributeKeyFeeder, msg.Delegate),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Operator),
		),
	})

	return &types.MsgDelegateFeedConsentResponse{}, nil
}
