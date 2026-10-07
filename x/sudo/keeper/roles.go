package keeper

import (
	"context"
	"fmt"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func (k Keeper) UpdateRoleMembers(goCtx context.Context, msg *sudo.MsgUpdateRoleMembers) (*sudo.MsgUpdateRoleMembersResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	state, err := k.Sudoers.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := k.CheckPermissions(msg.GetSigners()[0], ctx, ""); err != nil {
		return nil, err
	}
	index := -1
	for i, entry := range state.Roles {
		if entry.Role == msg.Role {
			index = i
			break
		}
	}
	if index < 0 {
		state.Roles = append(state.Roles, sudo.RoleMembers{Role: msg.Role})
		index = len(state.Roles) - 1
	}
	members := make(map[string]bool)
	for _, member := range state.Roles[index].Members {
		members[member] = true
	}
	for _, member := range msg.Add {
		members[member] = true
	}
	for _, member := range msg.Remove {
		delete(members, member)
	}
	state.Roles[index].Members = make([]string, 0, len(members))
	for member := range members {
		state.Roles[index].Members = append(state.Roles[index].Members, member)
	}
	if len(members) == 0 {
		state.Roles = append(state.Roles[:index], state.Roles[index+1:]...)
	}
	state.NormalizeRoles()
	k.Sudoers.Set(ctx, state)
	if err := ctx.EventManager().EmitTypedEvent(&sudo.EventUpdateSudoers{Sudoers: state, Action: "update_role_members"}); err != nil {
		return nil, err
	}
	return &sudo.MsgUpdateRoleMembersResponse{}, nil
}

// Migrate1To2 rewrites legacy state. Root keeps wire tag 1; the retired tag 2
// is skipped by the new protobuf decoder and never becomes a role grant.
func (k Keeper) Migrate1To2(ctx sdk.Context) error {
	state, err := k.Sudoers.Get(ctx)
	if err != nil {
		return fmt.Errorf("read legacy sudo state: %w", err)
	}
	state.Roles = []sudo.RoleMembers{}
	if err := state.Validate(); err != nil {
		return err
	}
	k.Sudoers.Set(ctx, state)
	return nil
}
