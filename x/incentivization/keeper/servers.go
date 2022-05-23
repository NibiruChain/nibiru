package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	"github.com/cosmos/cosmos-sdk/types/query"

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
	k Keeper
}

func (q queryServer) IncentivizationProgram(ctx context.Context, request *types.QueryIncentivizationProgramRequest) (*types.QueryIncentivizationProgramResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	program, err := q.k.IncentivizationProgramsState(sdkCtx).Get(request.Id)
	if err != nil {
		return nil, err
	}
	return &types.QueryIncentivizationProgramResponse{IncentivizationProgram: program}, nil
}

func (q queryServer) IncentivizationPrograms(ctx context.Context, request *types.QueryIncentivizationProgramsRequest) (*types.QueryIncentivizationProgramsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := prefix.NewStore(sdkCtx.KVStore(q.k.storeKey), incentivizationProgramObjectsNamespace)

	var programs []*types.IncentivizationProgram
	pageResp, err := query.Paginate(store, request.Pagination, func(key []byte, value []byte) error {
		bytes := store.Get(key)
		program := new(types.IncentivizationProgram)
		q.k.cdc.MustUnmarshal(bytes, program)
		programs = append(programs, program)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryIncentivizationProgramsResponse{
		IncentivizationPrograms: programs,
		Pagination:              pageResp,
	}, nil
}
