package keeper_test

import (
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/consensus/types"
)

func (s *KeeperTestSuite) TestGRPCQueryConsensusParams() {
	defaultConsensusParams := cmttypes.DefaultConsensusParams().ToProto()

	testCases := []struct {
		msg      string
		req      types.QueryParamsRequest
		malleate func()
		response types.QueryParamsResponse
		expPass  bool
	}{
		{
			"success",
			types.QueryParamsRequest{},
			func() {
				input := &types.MsgUpdateParams{
					Authority: s.consensusParamsKeeper.GetAuthority(),
					Block:     defaultConsensusParams.Block,
					Validator: defaultConsensusParams.Validator,
					Evidence:  defaultConsensusParams.Evidence,
				}
				s.msgServer.UpdateParams(s.ctx, input) //nolint:errcheck
			},
			types.QueryParamsResponse{
				Params: &tmproto.ConsensusParams{
					Block:     defaultConsensusParams.Block,
					Validator: defaultConsensusParams.Validator,
					Evidence:  defaultConsensusParams.Evidence,
					Version:   defaultConsensusParams.Version,
				},
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.ctx)

			res, err := s.queryClient.Params(ctx, &tc.req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(tc.response.Params, res.Params)
			} else {
				s.Require().Error(err)
				s.Require().Nil(res)
			}
		})
	}
}
