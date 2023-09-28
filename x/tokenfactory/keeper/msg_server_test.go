package keeper_test

import (
	"testing"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *TestSuite) TestCreateDenom() {
	_, addrs := testutil.PrivKeyAddressPairs(4)

	testCases := []struct {
		name     string
		txMsg    *types.MsgCreateDenom
		wantErr  string
		preHook  func(ctx sdk.Context, bapp *app.NibiruApp)
		postHook func(ctx sdk.Context, bapp *app.NibiruApp)
	}{
		{
			name:    "happy path",
			txMsg:   &types.MsgCreateDenom{Sender: addrs[0].String(), Subdenom: "nusd"},
			wantErr: "",
			preHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				allDenoms := bapp.TokenFactoryKeeper.Store.Denoms.
					Iterate(ctx, collections.Range[string]{}).Values()
				s.Len(allDenoms, 0)
			},
			postHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				allDenoms := bapp.TokenFactoryKeeper.Store.Denoms.
					Iterate(ctx, collections.Range[string]{}).Values()
				s.Len(allDenoms, 1)
				s.Equal(allDenoms[0].Subdenom, "nusd")
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			if tc.preHook != nil {
				tc.preHook(s.ctx, s.app)
			}

			resp, err := s.app.TokenFactoryKeeper.CreateDenom(
				sdk.WrapSDKContext(s.ctx), tc.txMsg,
			)

			if tc.wantErr != "" {
				s.Error(err)
				s.ErrorContains(err, tc.wantErr)
				return
			}

			s.NoError(err)
			want := types.TFDenom{
				Creator:  tc.txMsg.Sender,
				Subdenom: tc.txMsg.Subdenom,
			}.String()
			s.Equal(want, resp.NewTokenDenom)

			if tc.postHook != nil {
				tc.postHook(s.ctx, s.app)
			}

		})
	}
}
