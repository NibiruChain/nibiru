package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

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

func (m msgServer) CreateIncentivizationProgram(ctx context.Context, msg *types.MsgCreateIncentivizationProgram) (*types.MsgCreateIncentivizationProgramResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t := sdkCtx.BlockTime()
	if msg.StartTime != nil {
		t = *msg.StartTime
	}

	createdProgram, err := m.Keeper.CreateIncentivizationProgram(sdkCtx, msg.LpDenom, *msg.MinLockupDuration, t, msg.Epochs)
	if err != nil {
		return nil, err
	}

	// in case the user provided initial funds we fund the incentivization program
	if !msg.InitialFunds.IsZero() {
		addr, err := sdk.AccAddressFromBech32(msg.Sender)
		if err != nil {
			panic(err)
		}
		err = m.Keeper.FundIncentivizationProgram(sdkCtx, createdProgram.Id, addr, msg.InitialFunds)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgCreateIncentivizationProgramResponse{
		ProgramId: createdProgram.Id,
	}, nil
}

func (m msgServer) FundIncentivizationProgram(ctx context.Context, msg *types.MsgFundIncentivizationProgram) (*types.MsgFundIncentivizationProgramResponse, error) {
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	err = m.Keeper.FundIncentivizationProgram(sdk.UnwrapSDKContext(ctx), msg.Id, sender, msg.Funds)
	if err != nil {
		return nil, err
	}

	return &types.MsgFundIncentivizationProgramResponse{}, nil
}

func NewQueryServer(k Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	Keeper
}
