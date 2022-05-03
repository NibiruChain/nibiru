package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	v1 "github.com/NibiruChain/nibiru/x/perp/types/v1"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      sdk.StoreKey
	memKey        sdk.StoreKey
	ParamSubspace paramtypes.Subspace

	BankKeeper    v1.BankKeeper
	AccountKeeper v1.AccountKeeper
	pfk           v1.PriceKeeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", v1.ModuleName))
}

// GetModuleAccountBalance gets the airdrop coin balance of module account.
func (k Keeper) GetModuleAccountBalance(ctx sdk.Context, denom string) sdk.Coin {
	moduleAccAddr := k.AccountKeeper.GetModuleAddress(v1.ModuleName)
	return k.BankKeeper.GetBalance(ctx, moduleAccAddr, denom)
}

// GetParams get all parameters as types.Params
func (k *Keeper) GetParams(ctx sdk.Context) (params v1.Params) {
	k.ParamSubspace.GetParamSet(ctx, &params)
	return params
}

// SetParams set the params
func (k *Keeper) SetParams(ctx sdk.Context, params v1.Params) {
	k.ParamSubspace.SetParamSet(ctx, &params)
}
