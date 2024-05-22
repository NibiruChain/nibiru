package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

func (s *TestSuite) TestQueryModuleParams() {
	res, err := s.querier.Params(s.GoCtx(), &types.QueryParamsRequest{})
	s.NoError(err)
	s.Equal(res.Params, types.DefaultModuleParams())
}

func (s *TestSuite) createDenom(
	creator sdk.AccAddress,
	subdenom string,
) {
	msg := &types.MsgCreateDenom{
		Sender:   creator.String(),
		Subdenom: subdenom,
	}
	s.NoError(s.HandleMsg(msg))
}

func (s *TestSuite) TestQueryDenoms() {
	creator := testutil.AccAddress()
	denom := types.TFDenom{
		Creator:  creator.String(),
		Subdenom: "abc",
	}
	s.createDenom(creator, denom.Subdenom)
	s.createDenom(creator, "foobar")

	queryDenoms := func(creator string) (
		resp *types.QueryDenomsResponse, err error,
	) {
		return s.querier.Denoms(s.GoCtx(),
			&types.QueryDenomsRequest{
				Creator: creator,
			})
	}

	denomsResp, err := queryDenoms(denom.Creator)
	s.NoError(err)
	s.ElementsMatch(denomsResp.Denoms, []string{
		denom.Denom().String(),
		types.TFDenom{
			Creator:  denom.Creator,
			Subdenom: "foobar",
		}.Denom().String(),
	})

	denomsResp, err = queryDenoms("creator")
	s.NoError(err)
	s.Len(denomsResp.Denoms, 0)

	_, err = queryDenoms("")
	s.ErrorContains(err, "empty creator address")

	_, err = s.querier.Denoms(s.GoCtx(), nil)
	s.ErrorContains(err, "nil msg")
}

func (s *TestSuite) TestQueryDenomInfo() {
	s.SetupTest()
	creator := testutil.AccAddress()
	denom := types.TFDenom{
		Creator:  creator.String(),
		Subdenom: "abc",
	}
	s.createDenom(creator, denom.Subdenom)

	s.Run("case: nil msg", func() {
		_, err := s.querier.DenomInfo(s.GoCtx(),
			nil)
		s.ErrorContains(err, "nil msg")
	})

	s.Run("case: fail denom validation", func() {
		_, err := s.querier.DenomInfo(s.GoCtx(),
			&types.QueryDenomInfoRequest{
				Denom: "notadenom",
			})
		s.ErrorContains(err, "denom format error")
		_, err = s.app.TokenFactoryKeeper.QueryDenomInfo(s.ctx, "notadenom")
		s.Error(err)
	})

	s.Run("case: happy", func() {
		resp, err := s.querier.DenomInfo(s.GoCtx(),
			&types.QueryDenomInfoRequest{
				Denom: denom.Denom().String(),
			})
		s.NoError(err)
		s.Equal(creator.String(), resp.Admin)
		s.Equal(denom.DefaultBankMetadata(), resp.Metadata)
	})
}
