package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

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

// Ensure the interface is properly implemented at compile time
var _ sudotypes.MsgServer = Keeper{}

// EditSudoers adds or removes sudo contracts from state.
func (k Keeper) EditSudoers(
	goCtx context.Context, msg *sudotypes.MsgEditSudoers,
) (*sudotypes.MsgEditSudoersResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	switch msg.RootAction() {
	case sudotypes.AddContracts:
		return k.AddContracts(goCtx, msg)
	case sudotypes.RemoveContracts:
		return k.RemoveContracts(goCtx, msg)
	default:
		return nil, fmt.Errorf("invalid action type specified on msg: %s", msg)
	}
}

func (k Keeper) SenderHasPermission(sender string, root string) error {
	if sender != root {
		return fmt.Errorf(`message must be sent by root user. root: "%s", sender: "%s"`,
			root, sender,
		)
	}
	return nil
}

// ————————————————————————————————————————————————————————————————————————————
// Encoder for the Sudoers type
// ————————————————————————————————————————————————————————————————————————————

func SudoersValueEncoder(cdc codec.BinaryCodec) collections.ValueEncoder[sudotypes.Sudoers] {
	return collections.ProtoValueEncoder[sudotypes.Sudoers](cdc)
}

type Sudoers struct {
	Root      string          `json:"root"`
	Contracts set.Set[string] `json:"contracts"`
}

func (sudo Sudoers) String() string {
	return sudo.ToPb().String()
}

func (sudo Sudoers) ToPb() sudotypes.Sudoers {
	return sudotypes.Sudoers{
		Root:      sudo.Root,
		Contracts: sudo.Contracts.ToSlice(),
	}
}

func SudoersFromPb(pbSudoers sudotypes.Sudoers) Sudoers {
	return Sudoers{
		Root:      pbSudoers.Root,
		Contracts: set.New[string](pbSudoers.Contracts...),
	}
}

func SudoersToPb(sudo Sudoers) sudotypes.Sudoers {
	return sudo.ToPb()
}

// ————————————————————————————————————————————————————————————————————————————
// AddContracts
// ————————————————————————————————————————————————————————————————————————————

// AddContracts adds contract addresses to the sudoer set.
func (sudo *Sudoers) AddContracts(
	contracts []string,
) (out set.Set[string], err error) {
	for _, contractStr := range contracts {
		contract, err := sdk.AccAddressFromBech32(contractStr)
		if err != nil {
			return out, err
		}
		sudo.Contracts.Add(contract.String())
	}
	return sudo.Contracts, err
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
	err = k.SenderHasPermission(msg.Sender, sudoersBefore.Root)
	if err != nil {
		return
	}

	// Update state
	contracts, err := sudoersBefore.AddContracts(msg.Contracts)
	if err != nil {
		return
	}
	pbSudoers := SudoersToPb(Sudoers{Root: sudoersBefore.Root, Contracts: contracts})
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
	err = k.SenderHasPermission(msg.Sender, sudoers.Root)
	if err != nil {
		return
	}

	// Update state
	sudoers.RemoveContracts(msg.Contracts)
	pbSudoers = SudoersToPb(sudoers)
	k.Sudoers.Set(ctx, pbSudoers)

	msgResp = new(sudotypes.MsgEditSudoersResponse)
	return msgResp, ctx.EventManager().EmitTypedEvent(&sudotypes.EventUpdateSudoers{
		Sudoers: pbSudoers,
		Action:  msg.Action,
	})
}

func (sudo *Sudoers) RemoveContracts(contracts []string) {
	for _, contract := range contracts {
		sudo.Contracts.Remove(contract)
	}
}

// ————————————————————————————————————————————————————————————————————————————
// Helper Functions
// ————————————————————————————————————————————————————————————————————————————

// CheckPermissions Checks if a contract is contained within the set of sudo
// contracts defined in the x/sudo module. These smart contracts are able to
// execute certain permissioned functions.
func (k Keeper) CheckPermissions(
	contract sdk.AccAddress, ctx sdk.Context,
) error {
	contracts, err := k.GetSudoContracts(ctx)
	if err != nil {
		return err
	}
	hasPermission := set.New(contracts...).Has(contract.String())
	if !hasPermission {
		return fmt.Errorf(
			"insufficient permissions on smart contract: %s. The sudo contracts are: %s",
			contract, contracts,
		)
	}
	return nil
}

func (k Keeper) GetSudoContracts(ctx sdk.Context) (contracts []string, err error) {
	state, err := k.Sudoers.Get(ctx)
	return state.Contracts, err
}

// ————————————————————————————————————————————————————————————————————————————
// Setters - for use in tests
// ————————————————————————————————————————————————————————————————————————————

// SetSudoContracts overwrites the state. This function is a convenience
// function for testing with permissioned contracts in other modules..
func (k Keeper) SetSudoContracts(contracts []string, ctx sdk.Context) {
	k.Sudoers.Set(ctx, sudotypes.Sudoers{
		Root:      "",
		Contracts: contracts,
	})
}
