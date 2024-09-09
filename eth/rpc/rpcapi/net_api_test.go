package rpcapi_test

import (
	"github.com/NibiruChain/nibiru/v2/app/appconst"
)

func (s *NetworkSuite) TestNetNamespace() {
	api := s.val.EthRpc_NET
	s.Require().True(api.Listening())
	s.EqualValues(
		appconst.GetEthChainID(s.val.ClientCtx.ChainID).String(), api.Version())
	s.Equal(0, api.PeerCount())
}
