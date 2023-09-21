package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

func (s *TestSuite) TestQueryModuleParams() {
	res, err := s.queryClient.Params(s.GoCtx(), &types.QueryParamsRequest{})
	s.NoError(err)
	s.Equal(*res, types.DefaultModuleParams())
}
