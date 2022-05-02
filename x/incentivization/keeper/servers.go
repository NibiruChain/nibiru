package keeper

import (
	"context"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t := sdkCtx.BlockTime()
	if program.StartTime != nil {
		t = *program.StartTime
	}

	createdProgram, err := m.Keeper.CreateIncentivizationProgram(sdkCtx, program.LpDenom, *program.MinLockupDuration, t, program.Epochs)
	if err != nil {
		return nil, err
	}

	// in case the user provided initial funds we fund the incentivization program
	if !program.InitialFunds.IsZero() {
		addr, err := sdk.AccAddressFromBech32(program.Sender)
		if err != nil {
			panic(err)
		}
		err = m.Keeper.FundIncentivizationProgram(sdkCtx, createdProgram.Id, addr, program.InitialFunds)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgCreateIncentivizationProgramResponse{
		ProgramId: createdProgram.Id,
	}, nil
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
