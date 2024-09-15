package rpcapi_test

import (
	"github.com/NibiruChain/nibiru/v2/app/appconst"
)

func (s *NodeSuite) TestNetNamespace() {
	api := s.node.EthRpc_NET
	s.Require().True(api.Listening())
	s.EqualValues(
		appconst.GetEthChainID(s.node.ClientCtx.ChainID).String(), api.Version())
	s.Equal(0, api.PeerCount())
}
