package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// Ensure the interface is properly implemented at compile time
var _ sudo.MsgServer = (*Keeper)(nil)

// EditSudoers adds or removes sudo contracts from state.
func (k Keeper) EditSudoers(
	goCtx context.Context, msg *sudo.MsgEditSudoers,
) (*sudo.MsgEditSudoersResponse, error) {
	switch msg.RootAction() {
	case sudo.AddContracts:
		return k.AddContracts(goCtx, msg)
	case sudo.RemoveContracts:
		return k.RemoveContracts(goCtx, msg)
	default:
		return nil, fmt.Errorf("invalid action type specified on msg: %s", msg)
	}
}

func (k Keeper) ChangeRoot(
	goCtx context.Context,
	msg *sudo.MsgChangeRoot,
) (*sudo.MsgChangeRootResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

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

	err = k.CheckPermissions(msg.GetSigners()[0], ctx)
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

// ————————————————————————————————————————————————————————————————————————————
// Encoder for the Sudoers type
// ————————————————————————————————————————————————————————————————————————————

type Sudoers struct {
	Root      string          `json:"root"`
	Contracts set.Set[string] `json:"contracts"`
}

func (sudoers Sudoers) String() string {
	r := sudoers.ToPb()
	return r.String()
}

func (sudoers Sudoers) ToPb() sudo.Sudoers {
	return sudo.Sudoers{
		Root:      sudoers.Root,
		Contracts: sudoers.Contracts.ToSlice(),
	}
}

func SudoersFromPb(pbSudoers sudo.Sudoers) Sudoers {
	return Sudoers{
		Root:      pbSudoers.Root,
		Contracts: set.New[string](pbSudoers.Contracts...),
	}
}

// AddContracts adds contract addresses to the sudoer set.
func (sudoers *Sudoers) AddContracts(
	contracts []string,
) (out set.Set[string], err error) {
	for _, contractStr := range contracts {
		contract, err := sdk.AccAddressFromBech32(contractStr)
		if err != nil {
			return out, err
		}
		sudoers.Contracts.Add(contract.String())
	}
	return sudoers.Contracts, err
}

func (sudoers *Sudoers) RemoveContracts(contracts []string) {
	for _, contract := range contracts {
		sudoers.Contracts.Remove(contract)
	}
}
