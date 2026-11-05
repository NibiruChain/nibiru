package keeper

import (
	"context"
	"fmt"
	"slices"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/collections"

	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

type Keeper struct {
	Sudoers                collections.Item[sudo.Sudoers]
	ZeroGasActors          collections.Item[sudo.ZeroGasActors]
	WasmBlockHooksContract collections.Item[string]
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
		WasmBlockHooksContract: collections.NewItem(
			storeKey,
			sudo.NamespaceWasmBlockHooksContract,
			nutil.StringValueEncoder,
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

// EditWasmBlockHooksContract updates the optional Wasm contract address used by
// x/wasm to discover ABCI block hook dispatch plans.
func (k Keeper) EditWasmBlockHooksContract(
	goCtx context.Context, msg *sudo.MsgEditSudoers,
) (msgResp *sudo.MsgEditSudoersResponse, err error) {
	if msg.RootAction() != sudo.EditWasmBlockHooksContract {
		err = fmt.Errorf("invalid action type %s for msg edit wasm block hooks contract", msg.Action)
		return
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	pbSudoers, err := k.Sudoers.Get(ctx)
	if err != nil {
		return nil, err
	}
	err = k.senderHasPermission(msg.Sender, pbSudoers.Root)
	if err != nil {
		return nil, err
	}

	contractAddr, err := sudo.WasmBlockHooksContractFromMsgContracts(msg.Contracts)
	if err != nil {
		return nil, err
	}
	k.WasmBlockHooksContract.Set(ctx, contractAddr)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sudo.EventTypeWasmBlockHooksContractUpdate,
		sdk.NewAttribute(sudo.AttributeKeyWasmBlockHooksContract, contractAddr),
	))

	msgResp = new(sudo.MsgEditSudoersResponse)
	return msgResp, ctx.EventManager().EmitTypedEvent(&sudo.EventUpdateSudoers{
		Sudoers: pbSudoers,
		Action:  msg.Action,
	})
}

// CheckPermissions allows root or a member of role. An empty role is root-only.
func (k Keeper) CheckPermissions(actor sdk.AccAddress, ctx sdk.Context, role string) error {
	state, err := k.Sudoers.Get(ctx)
	if err != nil {
		return err
	}
	if actor.String() == state.Root {
		return nil
	}
	if role != "" {
		for _, entry := range state.Roles {
			if entry.Role == role && slices.Contains(entry.Members, actor.String()) {
				return nil
			}
		}
	}
	if role == "" {
		return sudo.ErrUnauthorized.Wrapf("address %s requires sudo root", actor)
	}
	return sudo.ErrUnauthorized.Wrapf("address %s requires root or role %q", actor, role)
}

// InitGenesis initializes the module's state from a provided genesis state JSON.
func (k Keeper) InitGenesis(ctx sdk.Context, genState sudo.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}
	if err := genState.Sudoers.NormalizeRoles(); err != nil {
		panic(err)
	}
	k.Sudoers.Set(ctx, genState.Sudoers)
	if genState.ZeroGasActors != nil {
		k.ZeroGasActors.Set(ctx, *genState.ZeroGasActors)
	}
	k.WasmBlockHooksContract.Set(ctx, genState.WasmBlockHooksContract)
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
		Sudoers:                pbSudoers,
		ZeroGasActors:          &zeroGasActors,
		WasmBlockHooksContract: k.WasmBlockHooksContract.GetOr(ctx, ""),
	}
}
