package keeper

import (
	"context"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
	sudokeeper "github.com/NibiruChain/nibiru/v2/x/sudo/keeper"
	sudotypes "github.com/NibiruChain/nibiru/v2/x/sudo/types"
)

type msgServer struct {
	Keeper
	SudoKeeper sudokeeper.Keeper
}

// NewMsgServerImpl returns an implementation of the oracle MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper, sudoKeeper sudokeeper.Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper, SudoKeeper: sudoKeeper}
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

	err = ctx.EventManager().EmitTypedEvent(&types.EventAggregatePrevote{
		Validator: msg.Validator,
		Feeder:    msg.Feeder,
	})
	return &types.MsgAggregateExchangeRatePrevoteResponse{}, err
}

func (ms msgServer) AggregateExchangeRateVote(
	goCtx context.Context, msg *types.MsgAggregateExchangeRateVote,
) (msgResp *types.MsgAggregateExchangeRateVoteResponse, err error) {
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
		return nil, sdkerrors.Wrapf(
			types.ErrHashVerificationFailed, "must be given %s not %s", aggregatePrevote.Hash, hash,
		)
	}

	// Move aggregate prevote to aggregate vote with given exchange rates
	ms.Keeper.Votes.Insert(
		ctx, valAddr, types.NewAggregateExchangeRateVote(exchangeRateTuples, valAddr),
	)
	_ = ms.Keeper.Prevotes.Delete(ctx, valAddr)

	priceTuples, err := types.NewExchangeRateTuplesFromString(msg.ExchangeRates)
	if err != nil {
		return
	}
	err = ctx.EventManager().EmitTypedEvent(&types.EventAggregateVote{
		Validator: msg.Validator,
		Feeder:    msg.Feeder,
		Prices:    priceTuples,
	})

	return &types.MsgAggregateExchangeRateVoteResponse{}, err
}

func (ms msgServer) DelegateFeedConsent(
	goCtx context.Context, msg *types.MsgDelegateFeedConsent,
) (*types.MsgDelegateFeedConsentResponse, error) {
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

	err = ctx.EventManager().EmitTypedEvent(&types.EventDelegateFeederConsent{
		Feeder:    msg.Delegate,
		Validator: msg.Operator,
	})

	return &types.MsgDelegateFeedConsentResponse{}, err
}

// EditOracleParams: gRPC tx msg for editing the oracle module params.
// [SUDO] Only callable by sudoers.
func (ms msgServer) EditOracleParams(goCtx context.Context, msg *types.MsgEditOracleParams) (*types.MsgEditOracleParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid address")
	}

	err = ms.SudoKeeper.CheckPermissions(sender, ctx)
	if err != nil {
		return nil, sudotypes.ErrUnauthorized
	}

	params, err := ms.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get oracle params error: %s", err.Error())
	}

	mergedParams := mergeOracleParams(msg, params)

	ms.Keeper.UpdateParams(ctx, mergedParams)

	return &types.MsgEditOracleParamsResponse{}, nil
}
