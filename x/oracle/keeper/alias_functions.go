package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// GetOracleAccount returns oracle ModuleAccount
func (k Keeper) GetOracleAccount(ctx sdk.Context) authtypes.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
}

// GetRewardsForPair retrieves the balance of the oracle module account
func (k Keeper) GetRewardsForPair(ctx sdk.Context, denom string) sdk.Coin {
	// TODO(mercilex): this logic needs to be redefined. https://github.com/NibiruChain/nibiru/issues/805
	if denom != common.DenomGov {
		return sdk.NewCoin("zero", sdk.ZeroInt())
	}
	acc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	return k.bankKeeper.GetBalance(ctx, acc.GetAddress(), denom)
}
