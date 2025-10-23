package keeper

import (
	"context"
	"fmt"

	"github.com/NibiruChain/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

type Keeper struct {
	Sudoers       collections.Item[sudo.Sudoers]
	ZeroGasActors collections.Item[sudo.ZeroGasActors]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey types.StoreKey,
) Keeper {
	return Keeper{
		Sudoers: collections.NewItem(
			storeKey,
			sudo.NamespaceSudoers,
			collections.ProtoValueEncoder[sudo.Sudoers](cdc),
		),
		ZeroGasActors: collections.NewItem(
			storeKey,
			sudo.NamespaceZeroGasActors,
			collections.ProtoValueEncoder[sudo.ZeroGasActors](cdc),
		),
	}
}

// Returns the root address of the sudo module.
func (k Keeper) GetRootAddr(ctx sdk.Context) (sdk.AccAddress, error) {
	sudoers, err := k.Sudoers.Get(ctx)
	if err != nil {
		return nil, err
	}

	addr, err := sdk.AccAddressFromBech32(sudoers.Root)
	if err != nil {
		return nil, err
	}

	return addr, nil
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
	goCtx context.Context, msg *sudo.MsgEditSudoers,
) (msgResp *sudo.MsgEditSudoersResponse, err error) {
	if msg.RootAction() != sudo.AddContracts {
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
	msgResp = new(sudo.MsgEditSudoersResponse)
	return msgResp, ctx.EventManager().EmitTypedEvent(&sudo.EventUpdateSudoers{
		Sudoers: pbSudoers,
		Action:  msg.Action,
	})
}

// ————————————————————————————————————————————————————————————————————————————
// RemoveContracts
// ————————————————————————————————————————————————————————————————————————————

func (k Keeper) RemoveContracts(
	goCtx context.Context, msg *sudo.MsgEditSudoers,
) (msgResp *sudo.MsgEditSudoersResponse, err error) {
	if msg.RootAction() != sudo.RemoveContracts {
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

	msgResp = new(sudo.MsgEditSudoersResponse)
	return msgResp, ctx.EventManager().EmitTypedEvent(&sudo.EventUpdateSudoers{
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

	hasPermission := set.New(contracts...).Has(contract.String()) || contract.String() == state.Root
	if !hasPermission {
		return fmt.Errorf(
			"%s: insufficient permissions on smart contract: %s. The sudo contracts are: %s",
			sudo.ErrUnauthorized, contract, contracts,
		)
	}
	return nil
}

// InitGenesis initializes the module's state from a provided genesis state JSON.
func (k Keeper) InitGenesis(ctx sdk.Context, genState sudo.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}
	k.Sudoers.Set(ctx, genState.Sudoers)
	if genState.ZeroGasActors != nil {
		k.ZeroGasActors.Set(ctx, *genState.ZeroGasActors)
	}
}

// ExportGenesis returns the module's exported genesis state.
// This fn assumes [Keeper.InitGenesis] has already been called.
func (k Keeper) ExportGenesis(ctx sdk.Context) *sudo.GenesisState {
	pbSudoers, err := k.Sudoers.Get(ctx)
	if err != nil {
		panic(err)
	}

	// Get ZeroGasActors, use default if not set
	zeroGasActors := k.ZeroGasActors.GetOr(ctx, sudo.DefaultZeroGasActors())

	return &sudo.GenesisState{
		Sudoers:       pbSudoers,
		ZeroGasActors: &zeroGasActors,
	}
}
