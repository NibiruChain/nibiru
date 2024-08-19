package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/v2/x/common/set"
	sudotypes "github.com/NibiruChain/nibiru/v2/x/sudo/types"
)

type MsgServer struct {
	keeper Keeper
}

func NewMsgServer(keeper Keeper) *MsgServer {
	return &MsgServer{keeper: keeper}
}

// Ensure the interface is properly implemented at compile time
var _ sudotypes.MsgServer = MsgServer{}

// EditSudoers adds or removes sudo contracts from state.
func (m MsgServer) EditSudoers(
	goCtx context.Context, msg *sudotypes.MsgEditSudoers,
) (*sudotypes.MsgEditSudoersResponse, error) {
	switch msg.RootAction() {
	case sudotypes.AddContracts:
		return m.keeper.AddContracts(goCtx, msg)
	case sudotypes.RemoveContracts:
		return m.keeper.RemoveContracts(goCtx, msg)
	default:
		return nil, fmt.Errorf("invalid action type specified on msg: %s", msg)
	}
}

func (m MsgServer) ChangeRoot(ctx context.Context, msg *sudotypes.MsgChangeRoot) (*sudotypes.MsgChangeRootResponse, error) {
	sdkContext := sdk.UnwrapSDKContext(ctx)

	pbSudoers, err := m.keeper.Sudoers.Get(sdkContext)
	if err != nil {
		return nil, fmt.Errorf("failed to get sudoers: %w", err)
	}

	err = m.validateRootPermissions(pbSudoers, msg)
	if err != nil {
		return nil, err
	}

	pbSudoers.Root = msg.NewRoot
	m.keeper.Sudoers.Set(sdkContext, pbSudoers)

	return &sudotypes.MsgChangeRootResponse{}, nil
}

func (m MsgServer) validateRootPermissions(pbSudoers sudotypes.Sudoers, msg *sudotypes.MsgChangeRoot) error {
	root, err := sdk.AccAddressFromBech32(pbSudoers.Root)
	if err != nil {
		return fmt.Errorf("failed to parse root address: %w", err)
	}

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("failed to parse sender address: %w", err)
	}

	if !root.Equals(sender) {
		return sudotypes.ErrUnauthorized
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
	r := sudo.ToPb()
	return r.String()
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

func (sudo *Sudoers) RemoveContracts(contracts []string) {
	for _, contract := range contracts {
		sudo.Contracts.Remove(contract)
	}
}
