package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"

	"github.com/NibiruChain/nibiru/v2/x/collections"

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
	add, err := sudo.NormalizeRoleMembers(msg.Add)
	if err != nil {
		return nil, err
	}
	remove, err := sudo.NormalizeRoleMembers(msg.Remove)
	if err != nil {
		return nil, err
	}
	if err := state.NormalizeRoles(); err != nil {
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
	for _, member := range add {
		members[member] = true
	}
	for _, member := range remove {
		delete(members, member)
	}
	state.Roles[index].Members = make([]string, 0, len(members))
	for member := range members {
		state.Roles[index].Members = append(state.Roles[index].Members, member)
	}
	if len(members) == 0 {
		state.Roles = append(state.Roles[:index], state.Roles[index+1:]...)
	}
	if err := state.NormalizeRoles(); err != nil {
		return nil, err
	}
	k.Sudoers.Set(ctx, state)
	if err := ctx.EventManager().EmitTypedEvent(&sudo.EventUpdateSudoers{Sudoers: state, Action: "update_role_members"}); err != nil {
		return nil, err
	}
	return &sudo.MsgUpdateRoleMembersResponse{}, nil
}

// Migrate1To2 preserves the broad legacy authority as the two scoped roles.
// Decode tag 2 before the current Sudoers decoder discards the retired field.
func (k Keeper) Migrate1To2(ctx sdk.Context) error {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, 0)
	raw := collections.Map[uint64, sudo.Sudoers](k.Sudoers).GetStore(ctx).Get(key)
	if raw == nil {
		return fmt.Errorf("legacy sudo state not found")
	}
	state, members, err := decodeLegacySudoers(raw)
	if err != nil {
		return fmt.Errorf("read legacy sudo state: %w", err)
	}
	members, err = sudo.NormalizeRoleMembers(members)
	if err != nil {
		return fmt.Errorf("legacy sudo members: %w", err)
	}
	if len(members) > 0 {
		state.Roles = []sudo.RoleMembers{
			{Role: sudo.RoleTFOper, Members: members},
			{Role: sudo.RoleChainParams, Members: append([]string(nil), members...)},
		}
	}
	if err := state.Validate(); err != nil {
		return err
	}
	if err := state.NormalizeRoles(); err != nil {
		return err
	}
	k.Sudoers.Set(ctx, state)
	return nil
}

func decodeLegacySudoers(raw []byte) (state sudo.Sudoers, members []string, err error) {
	for len(raw) > 0 {
		field, typ, n := protowire.ConsumeTag(raw)
		if n < 0 {
			return state, nil, protowire.ParseError(n)
		}
		raw = raw[n:]
		if field == 1 || field == 2 {
			if typ != protowire.BytesType {
				return state, nil, fmt.Errorf("legacy sudo field %d has wire type %d", field, typ)
			}
			value, n := protowire.ConsumeString(raw)
			if n < 0 {
				return state, nil, protowire.ParseError(n)
			}
			if field == 1 {
				state.Root = value
			} else {
				members = append(members, value)
			}
			raw = raw[n:]
		} else {
			n := protowire.ConsumeFieldValue(field, typ, raw)
			if n < 0 {
				return state, nil, protowire.ParseError(n)
			}
			raw = raw[n:]
		}
	}
	return state, members, nil
}
