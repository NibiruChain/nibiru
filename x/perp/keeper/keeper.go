package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      sdk.StoreKey
	memKey        sdk.StoreKey
	ParamSubspace paramtypes.Subspace

	BankKeeper    types.BankKeeper
	AccountKeeper types.AccountKeeper
	PriceKeeper   types.PriceKeeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetModuleAccountBalance gets the airdrop coin balance of module account.
func (k Keeper) GetModuleAccountBalance(ctx sdk.Context, denom string) sdk.Coin {
	moduleAccAddr := k.AccountKeeper.GetModuleAddress(types.ModuleName)
	return k.BankKeeper.GetBalance(ctx, moduleAccAddr, denom)
}

// GetParams get all parameters as types.Params
func (k *Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.ParamSubspace.GetParamSet(ctx, &params)
	return params
}

// SetParams set the params
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.ParamSubspace.SetParamSet(ctx, &params)
}

// NewKeeper Creates a new x/perp Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	paramSubspace paramtypes.Subspace,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	priceKeeper types.PriceKeeper,
) Keeper {
	// Ensure that the module account is set.
	if moduleAcc := accountKeeper.GetModuleAddress(types.ModuleName); moduleAcc == nil {
		panic("The x/perp module account has not been set")
	}

	// Set param.types.'KeyTable' if it has not already been set
	if !paramSubspace.HasKeyTable() {
		paramSubspace = paramSubspace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		memKey:        memKey,
		ParamSubspace: paramSubspace,

		AccountKeeper: accountKeeper,
		BankKeeper:    bankKeeper,
		PriceKeeper:   priceKeeper,
	}
}
