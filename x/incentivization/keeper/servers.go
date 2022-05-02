package keeper

import (
	"context"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
)

var (
	_ types.MsgServer = (*msgServer)(nil)
)

func NewMsgServer(k Keeper) types.MsgServer {
	return msgServer{k}
}

type msgServer struct {
	Keeper
}

func (m msgServer) CreateIncentivizationProgram(ctx context.Context, program *types.MsgCreateIncentivizationProgram) (*types.MsgCreateIncentivizationProgramResponse, error) {
	panic("implement me")
}

func (m msgServer) FundIncentivizationProgram(ctx context.Context, program *types.MsgFundIncentivizationProgram) (*types.MsgFundIncentivizationProgramResponse, error) {
	panic("implement me")
}

func NewQueryServer(k Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	Keeper
}
