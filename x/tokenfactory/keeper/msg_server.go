package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

var _ types.MsgServer = (*Keeper)(nil)

var errNilTxMsg error = grpcstatus.Errorf(grpccodes.InvalidArgument, "nil tx msg")

func (k Keeper) CreateDenom(
	goCtx context.Context, txMsg *types.MsgCreateDenom,
) (resp *types.MsgCreateDenomResponse, err error) {
	if txMsg == nil {
		return resp, errNilTxMsg
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err // ValidateBasic needs to be guaranteed for Wasm bindings
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	denom := types.TFDenom{
		Creator:  txMsg.Sender,
		Subdenom: txMsg.Subdenom,
	}
	err = k.Store.InsertDenom(ctx, denom)
	if err != nil {
		return resp, err
	}

	return &types.MsgCreateDenomResponse{
		NewTokenDenom: denom.String(),
	}, err
}

func (k Keeper) ChangeAdmin(
	goCtx context.Context, txMsg *types.MsgChangeAdmin,
) (resp *types.MsgChangeAdminResponse, err error) {
	if txMsg == nil {
		return resp, errNilTxMsg
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err // ValidateBasic needs to be guaranteed for Wasm bindings
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	authData, err := k.Store.GetDenomAuthorityMetadata(ctx, txMsg.Denom)
	if txMsg.Sender != authData.Admin {
		return resp, types.ErrInvalidSender.Wrapf(
			"only the current admin can set a new admin: current admin (%s), sender (%s)",
			authData.Admin, txMsg.Sender,
		)
	}

	authData.Admin = txMsg.NewAdmin
	k.Store.denomAdmins.Insert(ctx, txMsg.Denom, authData)

	return &types.MsgChangeAdminResponse{}, ctx.EventManager().EmitTypedEvent(
		&types.EventChangeAdmin{
			Denom:    txMsg.Denom,
			OldAdmin: txMsg.Sender,
			NewAdmin: txMsg.NewAdmin,
		})
}

func (k Keeper) UpdateModuleParams(
	goCtx context.Context, txMsg *types.MsgUpdateModuleParams,
) (resp *types.MsgUpdateModuleParamsResponse, err error) {
	if txMsg == nil {
		return resp, errNilTxMsg
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err // ValidateBasic needs to be guaranteed for Wasm bindings
	}

	if k.authority != txMsg.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.authority, txMsg.Authority)
	}

	if err := txMsg.Params.Validate(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.Store.ModuleParams.Set(ctx, txMsg.Params)
	return &types.MsgUpdateModuleParamsResponse{}, err
}
