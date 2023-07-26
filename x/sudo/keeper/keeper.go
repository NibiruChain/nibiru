package keeper

import (
	"context"
	"fmt"

	"github.com/NibiruChain/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/set"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
)

type Keeper struct {
	Sudoers collections.Item[sudotypes.Sudoers]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey types.StoreKey,
) Keeper {
	return Keeper{
		Sudoers: collections.NewItem(storeKey, 1, SudoersValueEncoder(cdc)),
	}
}

func (k Keeper) senderHasPermission(sender string, root string) error {
	if sender != root {
		return fmt.Errorf(`message must be sent by root user. root: "%s", sender: "%s"`,
			root, sender,
		)
	}
	return nil
}

// AddContracts executes a MsgEditSudoers message with action type
// "add_contracts". This adds contract addresses to the sudoer set.
func (k Keeper) AddContracts(
	goCtx context.Context, msg *sudotypes.MsgEditSudoers,
) (msgResp *sudotypes.MsgEditSudoersResponse, err error) {
	if msg.RootAction() != sudotypes.AddContracts {
		err = fmt.Errorf("invalid action type %s for msg add contracts", msg.Action)
		return
	}

	// Read state
	ctx := sdk.UnwrapSDKContext(goCtx)
	pbSudoersBefore, err := k.Sudoers.Get(ctx)
	if err != nil {
		return
	}
	sudoersBefore := SudoersFromPb(pbSudoersBefore)
	err = k.senderHasPermission(msg.Sender, sudoersBefore.Root)
	if err != nil {
		return
	}

	// Update state
	contracts, err := sudoersBefore.AddContracts(msg.Contracts)
	if err != nil {
		return
	}
	pbSudoers := Sudoers{Root: sudoersBefore.Root, Contracts: contracts}.ToPb()
	k.Sudoers.Set(ctx, pbSudoers)
	msgResp = new(sudotypes.MsgEditSudoersResponse)
	return msgResp, ctx.EventManager().EmitTypedEvent(&sudotypes.EventUpdateSudoers{
		Sudoers: pbSudoers,
		Action:  msg.Action,
	})
}

// ————————————————————————————————————————————————————————————————————————————
// RemoveContracts
// ————————————————————————————————————————————————————————————————————————————

func (k Keeper) RemoveContracts(
	goCtx context.Context, msg *sudotypes.MsgEditSudoers,
) (msgResp *sudotypes.MsgEditSudoersResponse, err error) {
	if msg.RootAction() != sudotypes.RemoveContracts {
		err = fmt.Errorf("invalid action type %s for msg add contracts", msg.Action)
		return
	}

	// Skip "msg.ValidateBasic" since this is a remove' operation. That means we
	// can only remove from state but can't write anything invalid that would
	// corrupt it.

	// Read state
	ctx := sdk.UnwrapSDKContext(goCtx)
	pbSudoers, err := k.Sudoers.Get(ctx)
	if err != nil {
		return
	}
	sudoers := SudoersFromPb(pbSudoers)
	err = k.senderHasPermission(msg.Sender, sudoers.Root)
	if err != nil {
		return
	}

	// Update state
	sudoers.RemoveContracts(msg.Contracts)
	pbSudoers = sudoers.ToPb()
	k.Sudoers.Set(ctx, pbSudoers)

	msgResp = new(sudotypes.MsgEditSudoersResponse)
	return msgResp, ctx.EventManager().EmitTypedEvent(&sudotypes.EventUpdateSudoers{
		Sudoers: pbSudoers,
		Action:  msg.Action,
	})
}

// CheckPermissions Checks if a contract is contained within the set of sudo
// contracts defined in the x/sudo module. These smart contracts are able to
// execute certain permissioned functions.
func (k Keeper) CheckPermissions(
	contract sdk.AccAddress, ctx sdk.Context,
) error {
	state, err := k.Sudoers.Get(ctx)
	if err != nil {
		return err
	}
	contracts := state.Contracts

	hasPermission := set.New(contracts...).Has(contract.String())
	if !hasPermission {
		return fmt.Errorf(
			"insufficient permissions on smart contract: %s. The sudo contracts are: %s",
			contract, contracts,
		)
	}
	return nil
}
