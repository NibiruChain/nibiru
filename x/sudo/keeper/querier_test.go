package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/sudo/keeper"
	"github.com/NibiruChain/nibiru/x/sudo/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestQuerySudoers(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state types.Sudoers
	}{
		{
			name: "happy 1",
			state: types.Sudoers{
				Root:      "alice",
				Contracts: []string{"contractA", "contractB"},
			},
		},

		{
			name: "happy 2 (empty)",
			state: types.Sudoers{
				Root:      "",
				Contracts: []string(nil),
			},
		},

		{
			name: "happy 3",
			state: types.Sudoers{
				Root:      "",
				Contracts: []string{"boop", "blap"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nibiru, ctx := setup()

			nibiru.SudoKeeper.Sudoers.Set(ctx, tc.state)

			req := new(types.QuerySudoersRequest)
			querier := keeper.NewQuerier(nibiru.SudoKeeper)
			resp, err := querier.QuerySudoers(
				sdk.WrapSDKContext(ctx), req,
			)
			require.NoError(t, err)

			outSudoers := resp.Sudoers
			require.EqualValues(t, tc.state, outSudoers)
		})
	}

	t.Run("nil request should error", func(t *testing.T) {
		nibiru, ctx := setup()
		var req *types.QuerySudoersRequest = nil
		querier := keeper.NewQuerier(nibiru.SudoKeeper)
		_, err := querier.QuerySudoers(
			sdk.WrapSDKContext(ctx), req,
		)
		require.Error(t, err)
	})
}
