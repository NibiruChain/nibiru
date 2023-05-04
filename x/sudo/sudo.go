package sudo

import (
	"context"
	"fmt"

	"github.com/NibiruChain/collections"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/sudo/pb"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type Keeper struct {
	Sudoers collections.Item[PbSudoers]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
) Keeper {
	return Keeper{
		Sudoers: collections.NewItem(storeKey, 1, SudoersValueEncoder(cdc)),
	}
}

var ROOT_ACTIONS = pb.ROOT_ACTIONS
var ROOT_ACTION = pb.ROOT_ACTION

func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		goCtx := sdk.WrapSDKContext(
			ctx.WithEventManager(sdk.NewEventManager()),
		)
		switch msg := msg.(type) {
		case *pb.MsgEditSudoers:
			res, err := k.EditSudoers(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)

		default:
			errMsg := fmt.Sprintf(
				"unrecognized %s message type: %T", pb.ModuleName, msg)
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	}
}

// Ensure the interface is properly implemented at compile time
var _ pb.MsgServer = Keeper{}

func (k Keeper) EditSudoers(
	goCtx context.Context, msg *pb.MsgEditSudoers,
) (*pb.MsgEditSudoersResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	switch msg.Action {
	case ROOT_ACTION.AddContracts:
		return k.AddContracts(goCtx, msg)
	case ROOT_ACTION.RemoveContracts:
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

func SudoersValueEncoder(cdc codec.BinaryCodec) collections.ValueEncoder[pb.Sudoers] {
	return collections.ProtoValueEncoder[pb.Sudoers](cdc)
}

type PbSudoers = pb.Sudoers

type Sudoers struct {
	Root      string          `json:"root"`
	Contracts set.Set[string] `json:"contracts"`
}

func (sudo Sudoers) String() string {
	return sudo.ToPb().String()
}

func (sudo Sudoers) ToPb() pb.Sudoers {
	return pb.Sudoers{
		Root:      sudo.Root,
		Contracts: sudo.Contracts.ToSlice(),
	}
}

func (sudo Sudoers) FromPb(pbSudoers pb.Sudoers) Sudoers {
	return Sudoers{
		Root:      pbSudoers.Root,
		Contracts: set.New[string](pbSudoers.Contracts...),
	}
}

func SudoersToPb(sudo Sudoers) pb.Sudoers {
	return sudo.ToPb()
}

// ————————————————————————————————————————————————————————————————————————————
// AddContracts
// ————————————————————————————————————————————————————————————————————————————

// Sudoers.AddContracts adds contract addresses to the sudoer set.
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

// Keeper.AddContracts executes a MsgEditSudoers message with action type
// "add_contracts". This adds contract addresses to the sudoer set.
func (k Keeper) AddContracts(
	goCtx context.Context, msg *pb.MsgEditSudoers,
) (msgResp *pb.MsgEditSudoersResponse, err error) {
	if msg.Action != ROOT_ACTION.AddContracts {
		err = fmt.Errorf("invalid action type %s for msg add contracts", msg.Action)
		return
	}

	// Read state
	ctx := sdk.UnwrapSDKContext(goCtx)
	pbSudoersBefore, err := k.Sudoers.Get(ctx)
	if err != nil {
		return
	}
	sudoersBefore := Sudoers{}.FromPb(pbSudoersBefore)
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
	msgResp = new(pb.MsgEditSudoersResponse)
	return msgResp, ctx.EventManager().EmitTypedEvent(&pb.EventUpdateSudoers{
		Sudoers: pbSudoers,
		Action:  msg.Action,
	})
}

// ————————————————————————————————————————————————————————————————————————————
// RemoveContracts
// ————————————————————————————————————————————————————————————————————————————

func (k Keeper) RemoveContracts(
	goCtx context.Context, msg *pb.MsgEditSudoers,
) (msgResp *pb.MsgEditSudoersResponse, err error) {
	if msg.Action != ROOT_ACTION.RemoveContracts {
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
	sudoers := Sudoers{}.FromPb(pbSudoers)
	err = k.SenderHasPermission(msg.Sender, sudoers.Root)
	if err != nil {
		return
	}

	// Update state
	sudoers.RemoveContracts(msg.Contracts)
	pbSudoers = SudoersToPb(sudoers)
	k.Sudoers.Set(ctx, pbSudoers)

	msgResp = new(pb.MsgEditSudoersResponse)
	return msgResp, ctx.EventManager().EmitTypedEvent(&pb.EventUpdateSudoers{
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
// Setters - for use in tests
// ————————————————————————————————————————————————————————————————————————————

func (k Keeper) GetSudoContracts(ctx sdk.Context) (contracts []string, err error) {
	state, err := k.Sudoers.Get(ctx)
	return state.Contracts, err
}

// SetSudoContracts overwrites the state. This function is a convenience
// function for testing with permissioned contracts in other modules..
func (k Keeper) SetSudoContracts(contracts []string, ctx sdk.Context) {
	k.Sudoers.Set(ctx, pb.Sudoers{
		Root:      "",
		Contracts: contracts,
	})
}
