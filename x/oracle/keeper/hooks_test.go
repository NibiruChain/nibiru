package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
)

func TestHooks_AfterEpochEnd(t *testing.T) {
	tests := []struct {
		name                   string
		initialFunds           sdk.Coins
		epochIdentifier        string
		expectedOracleBalances sdk.Coins
		expectedEFBalances     sdk.Coins
	}{
		{
			"happy path",
			sdk.NewCoins(
				sdk.NewCoin("coin1", sdk.NewInt(1000000000000000000)),
				sdk.NewCoin("coin2", sdk.NewInt(1000000000000000000)),
			),
			types.WeekEpochID,
			sdk.NewCoins(
				sdk.NewCoin("coin1", sdk.NewInt(50000000000000000)),
				sdk.NewCoin("coin2", sdk.NewInt(50000000000000000)),
			),
			sdk.NewCoins(
				sdk.NewCoin("coin1", sdk.NewInt(950000000000000000)),
				sdk.NewCoin("coin2", sdk.NewInt(950000000000000000)),
			),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			h := keeper.NewHooks(app.OracleKeeper, app.AccountKeeper, app.BankKeeper)

			err := testapp.FundModuleAccount(app.BankKeeper, ctx, perptypes.FeePoolModuleAccount, tt.initialFunds)
			require.NoError(t, err)

			h.AfterEpochEnd(ctx, tt.epochIdentifier, 0)

			account := app.AccountKeeper.GetModuleAccount(ctx, oracletypes.ModuleName)
			balances := app.BankKeeper.GetAllBalances(ctx, account.GetAddress())
			require.True(t, tt.expectedOracleBalances.IsEqual(balances))

			account = app.AccountKeeper.GetModuleAccount(ctx, perptypes.PerpEFModuleAccount)
			balances = app.BankKeeper.GetAllBalances(ctx, account.GetAddress())
			require.True(t, tt.expectedEFBalances.IsEqual(balances))
		})
	}
}
