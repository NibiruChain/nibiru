package ibcnibi

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	porttypes "github.com/cosmos/ibc-go/v3/modules/core/05-port/types"

	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// Keeper struct
type ICS4Keeper struct {
	cdc           codec.Codec
	storeKey      sdk.StoreKey
	paramstore    paramtypes.Subspace
	accountKeeper AccountKeeper
	bankKeeper    BankKeeper
	stakingKeeper StakingKeeper
	distrKeeper   DistrKeeper
	ics4Wrapper   porttypes.ICS4Wrapper
}

// NewKeeper returns keeper
func NewICS4Keeper(
	cdc codec.Codec,
	storeKey sdk.StoreKey,
	ps paramtypes.Subspace,
	ak AccountKeeper,
	bk BankKeeper,
	sk StakingKeeper,
	dk DistrKeeper,
) *ICS4Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		keyTable := paramtypes.NewKeyTable()
		ps = ps.WithKeyTable(keyTable)
	}

	return &ICS4Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		paramstore:    ps,
		accountKeeper: ak,
		bankKeeper:    bk,
		stakingKeeper: sk,
		distrKeeper:   dk,
	}
}

// SetICS4Wrapper sets the ICS4 wrapper to the keeper.
// It panics if already set
func (k *ICS4Keeper) SetICS4Wrapper(ics4Wrapper porttypes.ICS4Wrapper) {
	if k.ics4Wrapper != nil {
		panic("ICS4 wrapper already set")
	}

	k.ics4Wrapper = ics4Wrapper
}

var moduleName = "ibcnibi"

// Logger returns logger
func (k ICS4Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", moduleName))
}

// GetModuleAccount returns the module account for the claim module
func (k ICS4Keeper) GetModuleAccount(ctx sdk.Context) authtypes.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, moduleName)
}

// GetModuleAccountAddress gets the airdrop coin balance of module account
func (k ICS4Keeper) GetModuleAccountAddress() sdk.AccAddress {
	return k.accountKeeper.GetModuleAddress(moduleName)
}

// GetModuleAccountBalances gets the balances of module account that escrows the
// airdrop tokens
func (k ICS4Keeper) GetModuleAccountBalances(ctx sdk.Context) sdk.Coins {
	moduleAccAddr := k.GetModuleAccountAddress()
	return k.bankKeeper.GetAllBalances(ctx, moduleAccAddr)
}

var (
	_ transfertypes.ICS4Wrapper = ICS4Keeper{}
)

// IBC callbacks and transfer handlers

// SendPacket implements the ICS4Wrapper interface from the transfer module. It
// calls the underlying SendPacket function directly to move down the middleware
// stack. Without SendPacket, this module would be skipped, when sending packages
// from the transferKeeper to core IBC.
func (k ICS4Keeper) SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet exported.PacketI) error {
	return k.ics4Wrapper.SendPacket(ctx, channelCap, packet)
}

// WriteAcknowledgement implements the ICS4Wrapper interface from the transfer module.
// It calls the underlying WriteAcknowledgement function directly to move down the middleware stack.
func (k ICS4Keeper) WriteAcknowledgement(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet exported.PacketI, ack exported.Acknowledgement) error {
	return k.ics4Wrapper.WriteAcknowledgement(ctx, channelCap, packet, ack)
}
