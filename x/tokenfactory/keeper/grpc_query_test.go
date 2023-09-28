package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

func (s *TestSuite) TestQueryModuleParams() {
	res, err := s.querier.Params(s.GoCtx(), &types.QueryParamsRequest{})
	s.NoError(err)
	s.Equal(res.Params, types.DefaultModuleParams())
}
