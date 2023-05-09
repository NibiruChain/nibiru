package keeper

import (
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/NibiruChain/nibiru/x/epochs/types"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
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
	if epochIdentifier == types.WeekEpochID {
		account := h.accountKeeper.GetModuleAccount(ctx, perptypes.FeePoolModuleAccount)
		totalValidatorFees, totalRest := sdk.Coins{}, sdk.Coins{}

		balances := h.bankKeeper.GetAllBalances(ctx, account.GetAddress())
		for _, balance := range balances {
			validatorFees := balance.Amount.ToDec().Mul(sdk.MustNewDecFromStr("0.05")).TruncateInt()
			rest := balance.Amount.Sub(validatorFees)
			totalValidatorFees = append(totalValidatorFees, sdk.NewCoin(balance.Denom, validatorFees))
			totalRest = append(totalRest, sdk.NewCoin(balance.Denom, rest))
		}

		err := h.bankKeeper.SendCoinsFromModuleToModule(ctx, perptypes.FeePoolModuleAccount, perptypes.PerpEFModuleAccount, totalRest)
		if err != nil {
			panic(err)
		}

		err = h.k.AllocateRewards(
			ctx,
			perptypes.FeePoolModuleAccount,
			totalValidatorFees,
			1,
		)
		if err != nil {
			panic(err)
		}
	}
}

func (h Hooks) BeforeEpochStart(_ sdk.Context, _ string, _ uint64) {}
