package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/NibiruChain/nibiru/x/epochs/types"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	perptypes "github.com/NibiruChain/nibiru/x/perp/v2/types"
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
		params, err := h.k.Params.Get(ctx)
		if err != nil {
			h.k.Logger(ctx).Error("failed to get params", "error", err)
			return
		}

		account := h.accountKeeper.GetModuleAccount(ctx, perptypes.FeePoolModuleAccount)
		totalOracleRewards, totalRemainder := sdk.Coins{}, sdk.Coins{}

		balances := h.bankKeeper.GetAllBalances(ctx, account.GetAddress())
		for _, balance := range balances {
			oracleRewards := sdk.NewDecFromInt(balance.Amount).Mul(params.ValidatorFeeRatio).TruncateInt()
			remainder := balance.Amount.Sub(oracleRewards)

			if !oracleRewards.IsZero() {
				totalOracleRewards = append(totalOracleRewards, sdk.NewCoin(balance.Denom, oracleRewards))
			}

			if !remainder.IsZero() {
				totalRemainder = append(totalRemainder, sdk.NewCoin(balance.Denom, remainder))
			}
		}

		if !totalRemainder.IsZero() {
			err = h.bankKeeper.SendCoinsFromModuleToModule(ctx, perptypes.FeePoolModuleAccount, perptypes.PerpFundModuleAccount, totalRemainder)
			if err != nil {
				h.k.Logger(ctx).Error("Failed to send coins to perp ef module", "err", err)
			}
		}

		if !totalOracleRewards.IsZero() {
			err = h.k.AllocateRewards(
				ctx,
				perptypes.FeePoolModuleAccount,
				totalOracleRewards,
				1,
			)
			if err != nil {
				h.k.Logger(ctx).Error("Failed to allocate oracle rewards from perp fee pool", "err", err)
			}
		}
	}
}

func (h Hooks) BeforeEpochStart(_ sdk.Context, _ string, _ uint64) {}
