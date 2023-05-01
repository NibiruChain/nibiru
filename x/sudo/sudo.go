package sudo

import (
	"context"
	"fmt"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/sudo/pb"
	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type Sudoers struct {
	Root      string          `json:"root"`
	Contracts set.Set[string] `json:"contracts"`
}

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

func (k Keeper) EditSudoers(
	goCtx context.Context, msg *pb.MsgEditSudoers,
) (*pb.MsgEditSudoersResponse, error) {
	switch msg.Action {
	case ROOT_ACTION.AddContracts:
		return k.AddContracts(goCtx, msg)
	case ROOT_ACTION.RemoveContracts:
		return k.RemoveContracts(goCtx, msg)
	default:
		return nil, msg.ValidateBasic()
	}
}

var _ pb.MsgServer = Keeper{}

func NewMsgServerImpl(k Keeper) pb.MsgServer {
	return k
}

// TODO test
func (sudo *Sudoers) AddContracts(contracts []sdk.AccAddress) {
	for _, contract := range contracts {
		sudo.Contracts.Add(contract.String())
	}
}

func (sudo *Sudoers) TryAddContracts(contracts []string) error {
	for _, contractStr := range contracts {
		contract, err := sdk.AccAddressFromBech32(contractStr)
		if err != nil {
			return err
		}
		sudo.Contracts.Add(contract.String())
	}
	return nil
}

// TODO test
func (k Keeper) AddContracts(
	goCtx context.Context, msg *pb.MsgEditSudoers,
) (msgResp *pb.MsgEditSudoersResponse, err error) {
	if !(msg.Action == ROOT_ACTION.AddContracts) {
		err = fmt.Errorf("invalid action type %s for msg add contracts", msg.Action)
		return
	}

	err = msg.ValidateBasic()
	if err != nil {
		return
	}

	// Read state
	ctx := sdk.UnwrapSDKContext(goCtx)
	pbSudoers, err := k.Sudoers.Get(ctx)
	if err != nil {
		return
	}
	sudoers := Sudoers{}.FromPbSudoers(pbSudoers)

	// Update state
	err = sudoers.TryAddContracts(pbSudoers.Contracts)
	if err != nil {
		return
	}
	pbSudoers = SudoersToPb(sudoers)
	k.Sudoers.Set(ctx, pbSudoers)

	msgResp = new(pb.MsgEditSudoersResponse)
	return msgResp, ctx.EventManager().EmitTypedEvent(&pb.EventUpdateSudoers{
		Sudoers: pbSudoers,
		Action:  msg.Action,
	})
}

// TODO test
func (k Keeper) RemoveContracts(
	goCtx context.Context, msg *pb.MsgEditSudoers,
) (msgResp *pb.MsgEditSudoersResponse, err error) {
	if !(msg.Action == ROOT_ACTION.RemoveContracts) {
		err = fmt.Errorf("invalid action type %s for msg add contracts", msg.Action)
		return
	}

	// Skip "msg.ValidateBasic" since this is a remove' operation. That means we
	// can only remove state but can't write anything invalid.

	// Read state
	ctx := sdk.UnwrapSDKContext(goCtx)
	pbSudoers, err := k.Sudoers.Get(ctx)
	if err != nil {
		return
	}
	sudoers := Sudoers{}.FromPbSudoers(pbSudoers)

	// Update state
	sudoers.RemoveContracts(pbSudoers.Contracts)
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

// TODO test
// SetContracts overwrites with contracts set with the given contract addresses.
func (sudo *Sudoers) SetContracts(contracts []sdk.AccAddress) {
	sudo.Contracts = set.New[string]()
	for _, contract := range contracts {
		sudo.Contracts.Add(contract.String())
	}
}

// TODO test
// TrySetContracts overwrites with contracts set with the given contract
// addresses if they are valid Bech 32 public keys. Otherwise, it errors.
func (sudo *Sudoers) TrySetContracts(contracts []string) error {
	sudo.Contracts = set.New[string]()
	for _, contractStr := range contracts {
		contract, err := sdk.AccAddressFromBech32(contractStr)
		if err != nil {
			return err
		}
		sudo.Contracts.Add(contract.String())
	}
	return nil
}
