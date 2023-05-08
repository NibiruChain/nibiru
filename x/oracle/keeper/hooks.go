package keeper

import (
	"github.com/NibiruChain/nibiru/x/epochs/types"
	types2 "github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

var _ types.EpochHooks = Hooks{}

type Hooks struct {
	k             Keeper
	accountKeeper keeper.AccountKeeper
	bankKeeper    bankkeeper.Keeper
}

func NewHooks(k Keeper, accountKeeper keeper.AccountKeeper) *Hooks {
	return &Hooks{k: k, accountKeeper: accountKeeper}
}

func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, _ uint64) {
	if epochIdentifier == types.WeekEpochID {
		account := h.accountKeeper.GetModuleAccount(ctx, types2.FeePoolModuleAccount)

		balances := h.bankKeeper.GetAllBalances(ctx, account.GetAddress())
		for _, balance := range balances {
			validatorFees := balance.Amount.ToDec().Mul(sdk.MustNewDecFromStr("0.05")).TruncateInt()
			rest := balance.Amount.Sub(validatorFees)

			err := h.bankKeeper.SendCoinsFromModuleToModule(ctx, types2.FeePoolModuleAccount, types2.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(balance.Denom, rest)))
			if err != nil {
				panic(err)
			}

			h.k.AllocatePairRewards(ctx, types2.FeePoolModuleAccount, balance.Denom, validatorFees)
		}
	}
}

func (h Hooks) BeforeEpochStart(_ sdk.Context, _ string, _ uint64) {}
