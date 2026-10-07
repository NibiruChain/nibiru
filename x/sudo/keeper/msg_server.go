package keeper

import (
	"context"
	"fmt"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// Ensure the interface is properly implemented at compile time
var _ sudo.MsgServer = (*Keeper)(nil)

// EditSudoers configures the Wasm block-hook registry.
func (k Keeper) EditSudoers(
	goCtx context.Context, msg *sudo.MsgEditSudoers,
) (*sudo.MsgEditSudoersResponse, error) {
	switch msg.RootAction() {
	case sudo.AddContracts, sudo.RemoveContracts:
		return nil, fmt.Errorf("%s is retired; use UpdateRoleMembers", msg.Action)
	case sudo.EditWasmBlockHooksContract:
		return k.EditWasmBlockHooksContract(goCtx, msg)
	default:
		return nil, fmt.Errorf("invalid action type specified on msg: %s", msg)
	}
}

func (k Keeper) ChangeRoot(
	goCtx context.Context,
	msg *sudo.MsgChangeRoot,
) (*sudo.MsgChangeRootResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	pbSudoers, err := k.Sudoers.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sudoers: %w", err)
	}

	err = validateRootPermissions(pbSudoers, msg)
	if err != nil {
		return nil, err
	}

	pbSudoers.Root = msg.NewRoot
	k.Sudoers.Set(ctx, pbSudoers)

	return &sudo.MsgChangeRootResponse{}, nil
}

func validateRootPermissions(
	pbSudoers sudo.Sudoers,
	msg *sudo.MsgChangeRoot,
) error {
	root, err := sdk.AccAddressFromBech32(pbSudoers.Root)
	if err != nil {
		return fmt.Errorf("failed to parse root address: %w", err)
	}

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("failed to parse sender address: %w", err)
	}

	if !root.Equals(sender) {
		return sudo.ErrUnauthorized
	}

	return nil
}

func (k Keeper) EditZeroGasActors(
	goCtx context.Context,
	msg *sudo.MsgEditZeroGasActors,
) (*sudo.MsgEditZeroGasActorsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	err = k.CheckPermissions(msg.GetSigners()[0], ctx, "")
	if err != nil {
		return nil, err
	}

	actors := sudo.ZeroGasActors{}
	seenSenders := set.New[string]()
	seenContracts := set.New[string]()
	for _, sender := range msg.Actors.Senders {
		if seenSenders.Has(sender) {
			continue
		}
		actors.Senders = append(actors.Senders, sender)
		seenSenders.Add(sender)
	}
	for _, contract := range msg.Actors.Contracts {
		if seenContracts.Has(contract) {
			continue
		}
		actors.Contracts = append(actors.Contracts, contract)
		seenContracts.Add(contract)
	}
	seenAlwaysZeroGas := set.New[string]()
	for _, addr := range msg.Actors.AlwaysZeroGasContracts {
		if seenAlwaysZeroGas.Has(addr) {
			continue
		}
		actors.AlwaysZeroGasContracts = append(actors.AlwaysZeroGasContracts, addr)
		seenAlwaysZeroGas.Add(addr)
	}

	k.ZeroGasActors.Set(ctx, actors)

	return &sudo.MsgEditZeroGasActorsResponse{}, nil
}
