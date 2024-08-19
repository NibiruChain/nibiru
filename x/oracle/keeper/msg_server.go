package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/errors"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
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
func (ms msgServer) EditOracleParams(
	goCtx context.Context, msg *types.MsgEditOracleParams,
) (resp *types.MsgEditOracleParamsResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Stateless field validation is already performed in msg.ValidateBasic()
	// before the current scope is reached.
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	newParams, err := ms.Sudo().EditOracleParams(
		ctx, *msg, sender,
	)
	resp = &types.MsgEditOracleParamsResponse{
		NewParams: &newParams,
	}
	return resp, err
}
