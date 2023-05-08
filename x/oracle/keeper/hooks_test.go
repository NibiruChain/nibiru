package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	types2 "github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHooks_AfterEpochEnd(t *testing.T) {
	tests := []struct {
		name             string
		initialFunds     sdk.Coins
		epochIdentifier  string
		expectedBalances sdk.Coins
	}{
		{
			"happy path",
			sdk.NewCoins(
				sdk.NewCoin("coin1", sdk.NewInt(1000000000000000000)),
				sdk.NewCoin("coin2", sdk.NewInt(1000000000000000000)),
			),
			types.WeekEpochID,
			sdk.NewCoins(
				sdk.NewCoin("nfr", sdk.NewInt(1000000000000000000)),
			),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			h := keeper.NewHooks(app.OracleKeeper, app.AccountKeeper)
			h.AfterEpochEnd(ctx, tt.epochIdentifier, 0)

			err := testapp.FundModuleAccount(app.BankKeeper, ctx, types2.FeePoolModuleAccount, tt.initialFunds)
			require.NoError(t, err)

			h.BeforeEpochStart(ctx, tt.epochIdentifier, 0)

			account := app.AccountKeeper.GetModuleAccount(ctx, types2.FeePoolModuleAccount)
			balances := app.BankKeeper.GetAllBalances(ctx, account.GetAddress())
			require.Equal(t, nil, balances)
		})
	}
}
