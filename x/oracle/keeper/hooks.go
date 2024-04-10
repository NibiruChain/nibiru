package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/NibiruChain/nibiru/x/epochs/types"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

var _ types.EpochHooks = Hooks{}

// Return the wrapper struct
func (k Keeper) Hooks() Hooks {
	return Hooks{
		k,
		k.AccountKeeper,
		k.bankKeeper,
	}
}

type Hooks struct {
	k             Keeper
	accountKeeper oracletypes.AccountKeeper
	bankKeeper    oracletypes.BankKeeper
}

func NewHooks(k Keeper, accountKeeper keeper.AccountKeeper, bankKeeper bankkeeper.Keeper) *Hooks {
	return &Hooks{k: k, accountKeeper: accountKeeper, bankKeeper: bankKeeper}
}

func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, _ uint64) {
}

func (h Hooks) BeforeEpochStart(_ sdk.Context, _ string, _ uint64) {}
