package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/set"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
)

type MsgServer struct {
	keeper Keeper
}

func NewMsgServer(keeper Keeper) *MsgServer {
	return &MsgServer{keeper: keeper}
}

func (m MsgServer) ChangeRoot(ctx context.Context, root *sudotypes.MsgChangeRoot) (*sudotypes.MsgChangeRootResponse, error) {
	//TODO implement me
	panic("implement me")
}

// Ensure the interface is properly implemented at compile time
var _ sudotypes.MsgServer = MsgServer{}

// EditSudoers adds or removes sudo contracts from state.
func (m MsgServer) EditSudoers(
	goCtx context.Context, msg *sudotypes.MsgEditSudoers,
) (*sudotypes.MsgEditSudoersResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	switch msg.RootAction() {
	case sudotypes.AddContracts:
		return m.keeper.AddContracts(goCtx, msg)
	case sudotypes.RemoveContracts:
		return m.keeper.RemoveContracts(goCtx, msg)
	default:
		return nil, fmt.Errorf("invalid action type specified on msg: %s", msg)
	}
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
