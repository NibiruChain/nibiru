package keeper_test

import (
	testutilevents "github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *KeeperSuite) TestMsgCreateFunTokenFromCoin() {
	testCases := []struct {
		name    string
		denom   string
		sender  string
		wantErr string
	}{
		{
			name:    "happy: proper coin name",
			denom:   "tf/creator/usdt",
			sender:  "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
			wantErr: "",
		},
		{
			name:    "sad: empty coin name",
			denom:   "",
			sender:  "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
			wantErr: "invalid denom",
		},
		{
			name:    "sad: invalid sender",
			denom:   "tf/creator/usdt",
			sender:  "12345",
			wantErr: "invalid sender",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			resp, err := deps.Chain.EvmKeeper.CreateFunTokenFromCoin(
				deps.GoCtx(), &evm.MsgCreateFunTokenFromCoin{
					Denom:  tc.denom,
					Sender: tc.sender,
				},
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.Require().NotNil(resp)
			testutilevents.RequireContainsTypedEvent(
				s.T(),
				deps.Ctx,
				&evm.EventCreateFunTokenFromCoin{
					Creator:         tc.sender,
					Denom:           tc.denom,
					ContractAddress: resp.NewContractAddress,
				},
			)
		})
	}
}
