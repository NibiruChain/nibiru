package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

var _ types.MsgServer = (*Keeper)(nil)

func (k Keeper) AggregateExchangeRatePrevote(
	goCtx context.Context,
	msg *types.MsgAggregateExchangeRatePrevote,
) (*types.MsgAggregateExchangeRatePrevoteResponse, error) {
	return nil, types.ErrOracleDeprecated
}

func (k Keeper) AggregateExchangeRateVote(
	goCtx context.Context, msg *types.MsgAggregateExchangeRateVote,
) (msgResp *types.MsgAggregateExchangeRateVoteResponse, err error) {
	return nil, types.ErrOracleDeprecated
}

func (k Keeper) DelegateFeedConsent(
	goCtx context.Context, msg *types.MsgDelegateFeedConsent,
) (*types.MsgDelegateFeedConsentResponse, error) {
	return nil, types.ErrOracleDeprecated
}

// EditOracleParams: gRPC tx msg for editing the oracle module params.
// [SUDO] Only callable by sudoers.
func (k Keeper) EditOracleParams(goCtx context.Context, msg *types.MsgEditOracleParams) (*types.MsgEditOracleParamsResponse, error) {
	return nil, types.ErrOracleDeprecated
}
