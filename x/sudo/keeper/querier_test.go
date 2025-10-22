package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func (s *Suite) TestQuerySudoers() {
	for _, tc := range []struct {
		name  string
		state sudo.Sudoers
	}{
		{
			name: "happy 1",
			state: sudo.Sudoers{
				Root:      "alice",
				Contracts: []string{"contractA", "contractB"},
			},
		},

		{
			name: "happy 2 (empty)",
			state: sudo.Sudoers{
				Root:      "",
				Contracts: []string(nil),
			},
		},

		{
			name: "happy 3",
			state: sudo.Sudoers{
				Root:      "",
				Contracts: []string{"boop", "blap"},
			},
		},
	} {
		s.Run(tc.name, func() {
			_, k, ctx := setup()

			k.Sudoers.Set(ctx, tc.state)

			req := new(sudo.QuerySudoersRequest)
			resp, err := k.QuerySudoers(
				sdk.WrapSDKContext(ctx), req,
			)
			s.Require().NoError(err)

			outSudoers := resp.Sudoers
			s.Require().EqualValues(tc.state, outSudoers)
		})
	}

	s.Run("nil request should not error", func() {
		_, k, ctx := setup()
		var req *sudo.QuerySudoersRequest = nil
		_, err := k.QuerySudoers(
			sdk.WrapSDKContext(ctx), req,
		)
		s.Require().NoError(err)
	})
}
